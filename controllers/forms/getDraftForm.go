package controllers

import (
	"ccs-forms/db"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func GetDraftForm(c *gin.Context) {
	userID := c.GetInt64("userID")

	id, ok := draftFormID(c)
	if !ok {
		return
	}

	var form DraftForm
	err := db.Pool.QueryRow(c.Request.Context(), `
		SELECT id, title, description
		FROM draft_forms
		WHERE id = $1 AND author_id = $2`, id, userID).Scan(&form.ID, &form.Title, &form.Description)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Draft form not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get draft form"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"form": form})
}
