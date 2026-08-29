package controllers

import (
	"ccs-forms/db"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type PublishRequest struct {
	Deadline time.Time `json:"deadline"`
}

func PublishDraftForm(c *gin.Context) {
	var req PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if req.Deadline.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A deadline is required"})
		return
	}

	userID := c.GetInt64("userID")
	form_id, ok := draftFormID(c)
	if !ok {
		c.Status(http.StatusBadRequest)
		return
	}

	res, err := createCompleteForm(c)
	if err != nil {
		if writeRequestError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not prepare draft form for publishing"})
		return
	}

	var draft DraftForm
	err = db.Pool.QueryRow(c.Request.Context(),
		`SELECT id, title, description FROM draft_forms WHERE id=$1 AND author_id = $2`,
		form_id, userID,
	).Scan(&draft.ID, &draft.Title, &draft.Description)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "draft form not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "could not fetch the form",
		})
		return
	}

	_, err = db.Pool.Exec(c.Request.Context(),
		`INSERT INTO published_forms (id, title, description, author_id, deadline, structure)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		draft.ID, draft.Title, draft.Description, userID, req.Deadline, res,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "could not publish the form",
			"error":   err,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Succesfully Published the Form",
	})
}
