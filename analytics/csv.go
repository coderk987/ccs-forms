package analytics

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"ccs-forms/db"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

var errFormNotFound = errors.New("form not found")

// buildCSV turns one form's responses into a header row plus one row per
// response. Also returns the form title so the caller can name the file.
func buildCSV(ctx context.Context, formID, userID int64) (string, [][]string, error) {
	// the form must exist and belong to the caller
	var title string
	err := db.Pool.QueryRow(ctx, `
		SELECT title FROM published_forms
		WHERE id = $1 AND author_id = $2`, formID, userID).Scan(&title)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, errFormNotFound
	}
	if err != nil {
		return "", nil, err
	}

	// 1. questions, in form order -> these become the columns
	rows, err := db.Pool.Query(ctx, `
		SELECT q.id, q.title
		FROM questions q
		JOIN sections s ON s.id = q.section_id
		WHERE s.form_id = $1
		ORDER BY s.position, q.position`, formID)
	if err != nil {
		return "", nil, err
	}
	header := []string{"Response ID", "Submitted at", "Respondent"}
	col := make(map[int64]int) // question id -> column index
	for rows.Next() {
		var qid int64
		var qtitle string
		if err := rows.Scan(&qid, &qtitle); err != nil {
			rows.Close()
			return "", nil, err
		}
		col[qid] = len(header)
		header = append(header, qtitle)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return "", nil, err
	}

	// 2. responses, oldest first -> these become the rows
	rows, err = db.Pool.Query(ctx, `
		SELECT r.id, r."timestamp", u.name, u.gmail
		FROM responses r
		JOIN users u ON u.id = r.user_id
		WHERE r.form_id = $1
		ORDER BY r."timestamp"`, formID)
	if err != nil {
		return "", nil, err
	}
	out := [][]string{header}
	at := make(map[int64]int) // response id -> row index in out
	for rows.Next() {
		var rid int64
		var ts time.Time
		var name, email string
		if err := rows.Scan(&rid, &ts, &name, &email); err != nil {
			rows.Close()
			return "", nil, err
		}
		row := make([]string, len(header))
		row[0] = fmt.Sprint(rid)
		row[1] = ts.Format(time.RFC3339)
		row[2] = fmt.Sprintf("%s <%s>", name, email)
		at[rid] = len(out)
		out = append(out, row)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return "", nil, err
	}

	// 3. every answer, already flattened to the text that goes in the cell
	rows, err = db.Pool.Query(ctx, `
		SELECT a.response_id, a.question_id,
			CASE q.type
				WHEN 'checkbox' THEN COALESCE((
					SELECT string_agg(o.title, '; ' ORDER BY o.id)
					FROM checkbox_answers ca
					JOIN checkbox_options o ON o.id = ca.checkbox_option_id
					WHERE ca.answer_id = a.id), '')
				WHEN 'mcq' THEN COALESCE((
					SELECT o.title
					FROM mcq_options o
					WHERE o.question_id = a.question_id
						AND (a.payload->>'option_id' = o.id::text
							OR a.payload::text = o.id::text)), '')
				ELSE COALESCE(a.payload->>'text', a.payload->>'value',
					a.payload #>> '{}', '')
			END
		FROM answers a
		JOIN questions q ON q.id = a.question_id
		JOIN responses r ON r.id = a.response_id
		WHERE r.form_id = $1`, formID)
	if err != nil {
		return "", nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var rid, qid int64
		var value string
		if err := rows.Scan(&rid, &qid, &value); err != nil {
			return "", nil, err
		}
		i, okRow := at[rid]
		j, okCol := col[qid]
		if okRow && okCol {
			out[i][j] = value
		}
	}
	if err := rows.Err(); err != nil {
		return "", nil, err
	}

	return title, out, nil
}

var unsafeName = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func fileName(title string) string {
	slug := strings.Trim(unsafeName.ReplaceAllString(title, "_"), "_")
	if slug == "" {
		slug = "form"
	}
	return fmt.Sprintf("%s_responses_%s.csv", slug, time.Now().Format("2006-01-02"))
}

// ExportFormCSV streams the file straight to the browser as a download.
func ExportFormCSV(c *gin.Context) {
	userID := c.GetInt64("userID")
	formID, ok := publishedFormID(c)
	if !ok {
		return
	}

	title, records, err := buildCSV(c.Request.Context(), formID, userID)
	if errors.Is(err, errFormNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not build export"})
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition",
		fmt.Sprintf("attachment; filename=%q", fileName(title)))

	// BOM so Excel opens UTF-8 correctly
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(c.Writer)
	if err := w.WriteAll(records); err != nil {
		// headers are already sent, so nothing useful to return here
		return
	}
	w.Flush()
}

// SaveFormCSV writes the same data to a file on disk instead.
func SaveFormCSV(ctx context.Context, formID, userID int64, dir string) (string, error) {
	title, records, err := buildCSV(ctx, formID, userID)
	if err != nil {
		return "", err
	}

	path := dir + "/" + fileName(title)
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	f.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(f)
	if err := w.WriteAll(records); err != nil {
		return "", err
	}
	w.Flush()

	return path, w.Error()
}
