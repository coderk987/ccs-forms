package controllers

import (
	"ccs-forms/db"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CreateDraftForm(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req CreateDraftFormRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A title is required"})
		return
	}

	var form DraftForm
	err := db.Pool.QueryRow(c.Request.Context(), `
		INSERT INTO draft_forms (title, description, author_id)
		VALUES ($1, $2, $3)
		RETURNING id, title, description`, req.Title, req.Description, userID).Scan(
		&form.ID, &form.Title, &form.Description,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create draft form"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"form": form})
}
