package controllers

import (
	"ccs-forms/db"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func CreateResponse(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	formID, ok := publishedFormID(c)
	if !ok {
		return
	}

	var req CreateResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one answer is required"})
		return
	}

	ctx := c.Request.Context()

	// the form has to be live, still open and visible to this user
	var deadlinePassed bool
	err := db.Pool.QueryRow(ctx, `
		SELECT f.deadline IS NOT NULL AND f.deadline < NOW()
		FROM published_forms f
		WHERE f.id = $1
			AND (f.author_id = $2 OR EXISTS (
				SELECT 1 FROM view_permissions v
				WHERE v.form_id = f.id AND v.user_id = $2))`, formID, userID).Scan(&deadlinePassed)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get form"})
		return
	}
	if deadlinePassed {
		c.JSON(http.StatusForbidden, gin.H{"error": "Form is closed"})
		return
	}

	// every answered question must belong to this form
	questionIDs := make([]int64, 0, len(req.Answers))
	for _, answer := range req.Answers {
		questionIDs = append(questionIDs, answer.QuestionID)
	}

	var validQuestions int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT q.id)
		FROM questions q
		JOIN sections s ON s.id = q.section_id
		WHERE s.form_id = $1 AND q.id = ANY($2)`, formID, questionIDs).Scan(&validQuestions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not check answers"})
		return
	}
	if validQuestions != len(questionIDs) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Answers do not match the form questions"})
		return
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save response"})
		return
	}
	defer tx.Rollback(ctx)

	var response Response
	err = tx.QueryRow(ctx, `
		INSERT INTO responses (user_id, form_id)
		VALUES ($1, $2)
		RETURNING id, form_id, "timestamp"`, userID, formID).Scan(
		&response.ID, &response.FormID, &response.Timestamp,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save response"})
		return
	}

	for _, answer := range req.Answers {
		var answerID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO answers (response_id, question_id, payload)
			VALUES ($1, $2, $3)
			RETURNING id`, response.ID, answer.QuestionID, []byte(answer.Payload)).Scan(&answerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Could not save answers"})
			return
		}

		for _, optionID := range answer.CheckboxOptionIDs {
			// the option is only accepted if it belongs to the question it was sent with
			result, err := tx.Exec(ctx, `
				INSERT INTO checkbox_answers (checkbox_option_id, answer_id)
				SELECT o.id, $1
				FROM checkbox_options o
				WHERE o.id = $2 AND o.question_id = $3`, answerID, optionID, answer.QuestionID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save answers"})
				return
			}
			if result.RowsAffected() == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid checkbox option"})
				return
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save response"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"response": response})
}
