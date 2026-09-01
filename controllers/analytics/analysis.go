package controllers

import (
	"ccs-forms/db"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type OptionStat struct {
	OptionID int64  `json:"option_id"`
	Label    string `json:"label"`
	Count    int64  `json:"count"`
}

type QuestionStats struct {
	QuestionID int64        `json:"question_id"`
	Title      string       `json:"title"`
	Type       string       `json:"type"`
	Options    []OptionStat `json:"options,omitempty"`
}

type FormAnalytics struct {
	FormID         int64           `json:"form_id"`
	Title          string          `json:"title"`
	TotalResponses int64           `json:"total_responses"`
	Questions      []QuestionStats `json:"questions"`
}

func publishedFormID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form id"})
		return 0, false
	}
	return id, true
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
        SELECT f.title, COUNT(r.id)
        FROM published_forms f
        LEFT JOIN responses r ON r.form_id = f.id
        WHERE f.id = $1 AND f.author_id = $2
        GROUP BY f.id, f.title
    `, formID, userID).Scan(&out.Title, &out.TotalResponses)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get form"})
		return
	}

	questionsRows, err := db.Pool.Query(ctx, `
        SELECT q.id, q.title, q.type
        FROM questions q
        JOIN sections s ON s.id = q.section_id
        WHERE s.form_id = $1 AND q.type IN ('mcq', 'checkbox')
        ORDER BY s.position, q.position
    `, formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get questions"})
		return
	}
	defer questionsRows.Close()

	at := make(map[int64]int)

	for questionsRows.Next() {
		var q QuestionStats
		if err := questionsRows.Scan(&q.QuestionID, &q.Title, &q.Type); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read questions"})
			return
		}

		at[q.QuestionID] = len(out.Questions)
		out.Questions = append(out.Questions, q)
	}

	if err := questionsRows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read questions"})
		return
	}

	mcqRows, err := db.Pool.Query(ctx, `
        SELECT o.question_id, o.id, o.title, COUNT(DISTINCT a.id)
        FROM mcq_options o
        JOIN questions q ON q.id = o.question_id
        JOIN sections s ON s.id = q.section_id
        LEFT JOIN answers a
            ON a.question_id = o.question_id
            AND (
                (a.payload ? 'option_id' AND a.payload->>'option_id' = o.id::text)
                OR (jsonb_typeof(a.payload) = 'array' AND a.payload @> jsonb_build_array(o.id))
                OR (jsonb_typeof(a.payload) = 'number' AND a.payload::text = o.id::text)
            )
        WHERE s.form_id = $1 AND q.type = 'mcq'
        GROUP BY o.question_id, o.id, o.title
        ORDER BY o.question_id, o.id
    `, formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not count mcq answers"})
		return
	}
	if err := collectOptionCounts(mcqRows, out.Questions, at); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not count mcq answers"})
		return
	}

	checkboxRows, err := db.Pool.Query(ctx, `
        SELECT o.question_id, o.id, o.title, COUNT(DISTINCT a.id)
        FROM checkbox_options o
        JOIN questions q ON q.id = o.question_id
        JOIN sections s ON s.id = q.section_id
        LEFT JOIN answers a
            ON a.question_id = o.question_id
            AND (
                (jsonb_typeof(a.payload) = 'array' AND a.payload @> jsonb_build_array(o.id))
                OR (jsonb_typeof(a.payload) = 'number' AND a.payload::text = o.id::text)
            )
        WHERE s.form_id = $1 AND q.type = 'checkbox'
        GROUP BY o.question_id, o.id, o.title
        ORDER BY o.question_id, o.id
    `, formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not count checkbox answers"})
		return
	}
	if err := collectOptionCounts(checkboxRows, out.Questions, at); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not count checkbox answers"})
		return
	}

	c.JSON(http.StatusOK, out)
}

func collectOptionCounts(rows pgx.Rows, questions []QuestionStats, at map[int64]int) error {
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

		questions[i].Options = append(questions[i].Options, opt)
	}

	return rows.Err()
}
