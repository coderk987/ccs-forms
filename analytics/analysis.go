package analytics

import (
	"ccs-forms/db"
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type OptionStat struct {
	OptionID int64   `json:"option_id"`
	Label    string  `json:"label"`
	Count    int64   `json:"count"`
	Percent  float64 `json:"percent"`
}

type QuestionStats struct {
	QuestionID  int64        `json:"question_id"`
	Title       string       `json:"title"`
	Type        string       `json:"type"`
	Chart       string       `json:"chart"` // "pie" | "bar" | "list"
	Answered    int64        `json:"answered"`
	Options     []OptionStat `json:"options,omitempty"`
	TextAnswers []string     `json:"text_answers,omitempty"`
}

type DayCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type FormAnalytics struct {
	FormID         int64           `json:"form_id"`
	Title          string          `json:"title"`
	TotalResponses int64           `json:"total_responses"`
	Questions      []QuestionStats `json:"questions"`
	OverTime       []DayCount      `json:"responses_over_time"`
}

func publishedFormID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form id"})
		return 0, false
	}
	return id, true
}

func percent(count, total int64) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(float64(count)/float64(total)*1000) / 10
}

func GetFormAnalytics(c *gin.Context) {
	userID := c.GetInt64("userID")
	formID, ok := publishedFormID(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()

	out := FormAnalytics{FormID: formID}
	err := db.Pool.QueryRow(ctx, `
		SELECT f.title, (SELECT COUNT(*) FROM responses r WHERE r.form_id = f.id)
		FROM published_forms f
		WHERE f.id = $1 AND f.author_id = $2`, formID, userID).Scan(&out.Title, &out.TotalResponses)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get form"})
		return
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT q.id, q.title, q.type
		FROM questions q
		JOIN sections s ON s.id = q.section_id
		WHERE s.form_id = $1
		ORDER BY s.position, q.position`, formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get questions"})
		return
	}
	at := make(map[int64]int)
	for rows.Next() {
		var q QuestionStats
		if err := rows.Scan(&q.QuestionID, &q.Title, &q.Type); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read questions"})
			return
		}
		switch q.Type {
		case "mcq":
			q.Chart = "pie"
		case "checkbox":
			q.Chart = "bar"
		default:
			q.Chart = "list"
		}
		at[q.QuestionID] = len(out.Questions)
		out.Questions = append(out.Questions, q)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read questions"})
		return
	}
	rows, err = db.Pool.Query(ctx, `
		SELECT a.question_id, COUNT(*)
		FROM answers a
		JOIN responses r ON r.id = a.response_id
		WHERE r.form_id = $1
		GROUP BY a.question_id`, formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not count answers"})
		return
	}
	for rows.Next() {
		var qid, n int64
		if err := rows.Scan(&qid, &n); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not count answers"})
			return
		}
		if i, ok := at[qid]; ok {
			out.Questions[i].Answered = n
		}
	}
	rows.Close()

	// 4. mcq -> one slice per option (LEFT JOIN so unpicked options stay at 0)
	rows, err = db.Pool.Query(ctx, `
		SELECT o.question_id, o.id, o.title, COUNT(a.id)
		FROM mcq_options o
		JOIN questions q ON q.id = o.question_id
		JOIN sections s ON s.id = q.section_id
		LEFT JOIN answers a
			ON a.question_id = o.question_id
			AND (a.payload->>'option_id' = o.id::text OR a.payload::text = o.id::text)
			AND a.response_id IN (SELECT id FROM responses WHERE form_id = $1)
		WHERE s.form_id = $1
		GROUP BY o.question_id, o.id, o.title
		ORDER BY o.question_id, o.id`, formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not count mcq answers"})
		return
	}
	if err := collectOptions(rows, out.Questions, at); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not count mcq answers"})
		return
	}

	rows, err = db.Pool.Query(ctx, `
		SELECT o.question_id, o.id, o.title,
			COUNT(ca.answer_id) FILTER (WHERE r.form_id = $1)
		FROM checkbox_options o
		JOIN questions q ON q.id = o.question_id
		JOIN sections s ON s.id = q.section_id
		LEFT JOIN checkbox_answers ca ON ca.checkbox_option_id = o.id
		LEFT JOIN answers a ON a.id = ca.answer_id
		LEFT JOIN responses r ON r.id = a.response_id
		WHERE s.form_id = $1
		GROUP BY o.question_id, o.id, o.title
		ORDER BY o.question_id, o.id`, formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not count checkbox answers"})
		return
	}
	if err := collectOptions(rows, out.Questions, at); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not count checkbox answers"})
		return
	}

	rows, err = db.Pool.Query(ctx, `
		SELECT a.question_id,
			COALESCE(a.payload->>'text', a.payload->>'value', a.payload #>> '{}')
		FROM answers a
		JOIN responses r ON r.id = a.response_id
		JOIN questions q ON q.id = a.question_id
		WHERE r.form_id = $1 AND q.type = 'text'
		ORDER BY r."timestamp" DESC
		LIMIT 500`, formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get text answers"})
		return
	}
	for rows.Next() {
		var qid int64
		var text *string
		if err := rows.Scan(&qid, &text); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read text answers"})
			return
		}
		if i, ok := at[qid]; ok && text != nil && *text != "" {
			out.Questions[i].TextAnswers = append(out.Questions[i].TextAnswers, *text)
		}
	}
	rows.Close()

}

// shared by the mcq and checkbox queries - both return the same four columns
func collectOptions(rows pgx.Rows, questions []QuestionStats, at map[int64]int) error {
	defer rows.Close()
	for rows.Next() {
		var qid int64
		var opt OptionStat
		if err := rows.Scan(&qid, &opt.OptionID, &opt.Label, &opt.Count); err != nil {
			return err
		}
		i, ok := at[qid]
		if !ok {
			continue
		}
		opt.Percent = percent(opt.Count, questions[i].Answered)
		questions[i].Options = append(questions[i].Options, opt)
	}
	return rows.Err()
}
