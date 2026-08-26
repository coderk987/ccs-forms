package controllers

import (
	"ccs-forms/db"
	"net/http"

	"github.com/gin-gonic/gin"
)

func DeleteDraftForm(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := draftFormID(c)
	if !ok {
		return
	}

	result, err := db.Pool.Exec(c.Request.Context(), `
		DELETE FROM draft_forms WHERE id = $1 AND author_id = $2`, id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete draft form"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Draft form not found"})
		return
	}

	c.Status(http.StatusNoContent)
}
