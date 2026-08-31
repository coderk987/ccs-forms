package controllers

import (
	"ccs-forms/db"
	"net/http"

	"github.com/gin-gonic/gin"
)

func DeleteDraftForm(c *gin.Context) {
	userID := c.GetInt64("userID")

	id, ok := draftFormID(c)
	if !ok {
		return
	}

	var published bool
	err := db.Pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS (SELECT 1 FROM published_forms WHERE id = $1)`, id,
	).Scan(&published)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not check form status"})
		return
	}
	if published {
		c.JSON(http.StatusConflict, gin.H{"error": "Published forms cannot be deleted"})
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
