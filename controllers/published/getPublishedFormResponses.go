package controllers

import (
	"ccs-forms/db"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func GetPublishedFormResponses(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := publishedFormID(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()

	var formID int64
	err := db.Pool.QueryRow(ctx, `
		SELECT id FROM published_forms WHERE id = $1 AND author_id = $2`, id, userID).Scan(&formID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Published form not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get published form"})
		return
	}

	responseRows, err := db.Pool.Query(ctx, `
		SELECT r.id, r.user_id, u.name, u.gmail, r."timestamp"
		FROM responses r
		JOIN users u ON u.id = r.user_id
		WHERE r.form_id = $1
		ORDER BY r."timestamp" DESC, r.id DESC`, formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get responses"})
		return
	}
	defer responseRows.Close()

	responses := []FormResponse{}
	for responseRows.Next() {
		var response FormResponse
		response.Answers = []Answer{}
		if err := responseRows.Scan(
			&response.ID, &response.UserID, &response.Name, &response.Gmail, &response.Timestamp,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read responses"})
			return
		}
		responses = append(responses, response)
	}
	if responseRows.Err() != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read responses"})
		return
	}

	if len(responses) == 0 {
		c.JSON(http.StatusOK, gin.H{"responses": responses})
		return
	}

	responseIDs := make([]int64, 0, len(responses))
	byID := map[int64]*FormResponse{}
	for i := range responses {
		responseIDs = append(responseIDs, responses[i].ID)
		byID[responses[i].ID] = &responses[i]
	}

	answerRows, err := db.Pool.Query(ctx, `
		SELECT a.response_id, a.question_id, a.payload,
			COALESCE(array_agg(ca.checkbox_option_id)
				FILTER (WHERE ca.checkbox_option_id IS NOT NULL), '{}') AS checkbox_option_ids
		FROM answers a
		LEFT JOIN checkbox_answers ca ON ca.answer_id = a.id
		WHERE a.response_id = ANY($1)
		GROUP BY a.id
		ORDER BY a.id`, responseIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get answers"})
		return
	}
	defer answerRows.Close()

	for answerRows.Next() {
		var answer Answer
		var responseID int64
		if err := answerRows.Scan(
			&responseID, &answer.QuestionID, &answer.Payload, &answer.CheckboxOptionIDs,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read answers"})
			return
		}
		if response, ok := byID[responseID]; ok {
			response.Answers = append(response.Answers, answer)
		}
	}
	if answerRows.Err() != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read answers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"responses": responses})
}
