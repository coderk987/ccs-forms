package controllers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
	questionTypes := make(map[int64]string)
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

	// The response structure is the submission snapshot. It must be used for
	// values instead of answers.question_id, because published forms retain a
	// JSON snapshot while the draft question rows may later change.
	rows, err = db.Pool.Query(ctx, `
		SELECT q.id, q.type
		FROM questions q
		JOIN sections s ON s.id = q.section_id
		WHERE s.form_id = $1`, formID)
	if err != nil {
		return "", nil, err
	}
	for rows.Next() {
		var qid int64
		var qtype string
		if err := rows.Scan(&qid, &qtype); err != nil {
			rows.Close()
			return "", nil, err
		}
		questionTypes[qid] = qtype
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", nil, err
	}
	rows.Close()

	optionLabels, err := csvOptionLabels(ctx, formID)
	if err != nil {
		return "", nil, err
	}

	// 2. responses, oldest first -> these become the rows. Structure contains
	// answer values keyed by question ID, exactly as submitted by the client.
	rows, err = db.Pool.Query(ctx, `
		SELECT r.id, r."timestamp", u.name, u.gmail, r.structure
		FROM responses r
		JOIN users u ON u.id = r.user_id
		WHERE r.form_id = $1
		ORDER BY r."timestamp"`, formID)
	if err != nil {
		return "", nil, err
	}
	out := [][]string{header}
	for rows.Next() {
		var rid int64
		var ts time.Time
		var name, email string
		var structure json.RawMessage
		if err := rows.Scan(&rid, &ts, &name, &email, &structure); err != nil {
			rows.Close()
			return "", nil, err
		}
		row := make([]string, len(header))
		row[0] = fmt.Sprint(rid)
		row[1] = ts.Format(time.RFC3339)
		row[2] = fmt.Sprintf("%s <%s>", name, email)
		if err := fillCSVAnswers(row, structure, col, questionTypes, optionLabels); err != nil {
			rows.Close()
			return "", nil, err
		}
		out = append(out, row)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return "", nil, err
	}

	return title, out, nil
}

func csvOptionLabels(ctx context.Context, formID int64) (map[int64]map[int64]string, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT q.id, o.id, o.title FROM mcq_options o
		JOIN questions q ON q.id = o.question_id
		JOIN sections s ON s.id = q.section_id
		WHERE s.form_id = $1
		UNION ALL
		SELECT q.id, o.id, o.title FROM checkbox_options o
		JOIN questions q ON q.id = o.question_id
		JOIN sections s ON s.id = q.section_id
		WHERE s.form_id = $1`, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	labels := make(map[int64]map[int64]string)
	for rows.Next() {
		var qid int64
		var id int64
		var label string
		if err := rows.Scan(&qid, &id, &label); err != nil {
			return nil, err
		}
		if labels[qid] == nil {
			labels[qid] = make(map[int64]string)
		}
		labels[qid][id] = label
	}
	return labels, rows.Err()
}

func fillCSVAnswers(row []string, structure json.RawMessage, columns map[int64]int, types map[int64]string, labels map[int64]map[int64]string) error {
	var answers map[string]json.RawMessage
	if err := json.Unmarshal(structure, &answers); err != nil {
		return err
	}
	for key, raw := range answers {
		var qid int64
		if _, err := fmt.Sscan(key, &qid); err != nil {
			continue
		}
		column, ok := columns[qid]
		if !ok {
			continue
		}
		value, err := formatCSVAnswer(raw, types[qid], labels[qid])
		if err != nil {
			return err
		}
		row[column] = value
	}
	return nil
}

func formatCSVAnswer(raw json.RawMessage, questionType string, labels map[int64]string) (string, error) {
	if questionType == "mcq" || questionType == "checkbox" {
		var ids []json.RawMessage
		if len(raw) > 0 && raw[0] == '[' {
			if err := json.Unmarshal(raw, &ids); err != nil {
				return "", err
			}
		} else {
			ids = []json.RawMessage{raw}
		}
		values := make([]string, 0, len(ids))
		for _, idRaw := range ids {
			var id int64
			if err := json.Unmarshal(idRaw, &id); err != nil {
				var option struct {
					ID int64 `json:"option_id"`
				}
				if json.Unmarshal(idRaw, &option) != nil {
					return "", err
				}
				id = option.ID
			}
			if label, ok := labels[id]; ok {
				values = append(values, label)
			}
		}
		return strings.Join(values, "; "), nil
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, nil
	}
	var value struct {
		Text  string `json:"text"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw, &value); err == nil {
		if value.Text != "" {
			return value.Text, nil
		}
		return value.Value, nil
	}
	return string(raw), nil
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

	path := filepath.Join(dir, fileName(title))
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := f.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return "", err
	}

	w := csv.NewWriter(f)
	if err := w.WriteAll(records); err != nil {
		return "", err
	}
	w.Flush()

	return path, w.Error()
}
