package controllers

import (
	"ccs-forms/db"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type UpdatePublishedFormRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Deadline    *time.Time `json:"deadline"`
}

func UpdatePublishedForm(c *gin.Context) {
	userID := c.GetInt64("userID")

	id, ok := draftFormID(c)
	if !ok {
		return
	}

	var req UpdatePublishedFormRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Title == nil && req.Description == nil && req.Deadline == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "At least one of email, description, or deadline must be provided",
		})
		return
	}

	var form PublishedForm
	err := db.Pool.QueryRow(c.Request.Context(), `
		UPDATE draft_forms SET title = COALESCE($1, title), description = COALESCE($2, description), deadline = COALESCE($3, deadline)
		WHERE id = $4 AND author_id = $5
		RETURNING *`,
		req.Title, req.Description, req.Deadline, id, userID,
	).Scan(&form)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Published form not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update Published form"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"form": form,
	})
}
