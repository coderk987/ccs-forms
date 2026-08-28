package controllers

import (
	"ccs-forms/db"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type UpdateDraftFormRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

func UpdateDraftForm(c *gin.Context) {
	userID := c.GetInt64("userID")

	id, ok := draftFormID(c)
	if !ok {
		return
	}

	var req UpdateDraftFormRequest
	if err := c.ShouldBindJSON(&req); err != nil || (req.Title == nil && req.Description == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide a title or description to update"})
		return
	}

	var form DraftForm
	err := db.Pool.QueryRow(c.Request.Context(), `
		UPDATE draft_forms
		SET title = COALESCE($1, title), description = COALESCE($2, description)
		WHERE id = $3 AND author_id = $4
		RETURNING id, title, description`, req.Title, req.Description, id, userID).Scan(
		&form.ID, &form.Title, &form.Description,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Draft form not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update draft form"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"form": form})
}
