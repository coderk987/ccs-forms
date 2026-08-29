package controllers

import (
	"ccs-forms/db"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type responseRow struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	FormID    int64  `json:"form_id"`
	Timestamp string `json:"timestamp"`
}

type responseAnswerRow struct {
	ID         int64           `db:"id" json:"id"`
	QuestionID int64           `db:"question_id" json:"question_id"`
	Payload    json.RawMessage `db:"payload" json:"payload"`
	Structure  json.RawMessage `db:"structure" json:"structure"`
}

func GetResponse(c *gin.Context) {
	userID := c.GetInt64("userID")
	id, ok := responseID(c)
	if !ok {
		return
	}

	var response responseRow
	err := db.Pool.QueryRow(c.Request.Context(), `
		SELECT r.id, r.user_id, r.form_id, r."timestamp"::text
		FROM responses r
		JOIN published_forms f ON f.id = r.form_id
		WHERE r.id = $1
			AND (
				f.author_id = $2
				OR EXISTS (
					SELECT 1
					FROM view_permissions v
					WHERE v.form_id = f.id AND v.user_id = $2
				)
			)`,
		id, userID,
	).Scan(&response.ID, &response.UserID, &response.FormID, &response.Timestamp)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Response not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get response"})
		return
	}

	rows, err := db.Pool.Query(c.Request.Context(), `
		SELECT id, question_id, payload, structure
		FROM answers
		WHERE response_id = $1
		ORDER BY id ASC`,
		id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get response answers"})
		return
	}
	defer rows.Close()

	answers, err := pgx.CollectRows(rows, pgx.RowToStructByName[responseAnswerRow])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read response answers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"response": response,
		"answers":  answers,
	})
}
