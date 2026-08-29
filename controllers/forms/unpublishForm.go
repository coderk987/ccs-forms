package controllers

import (
	"net/http"

	"ccs-forms/db"

	"github.com/gin-gonic/gin"
)

func UnpublishForm(c *gin.Context) {
	userID := c.GetInt64("userID")
	id, ok := draftFormID(c)
	if !ok {
		return
	}

	result, err := db.Pool.Exec(c.Request.Context(),
		`DELETE FROM published_forms WHERE id = $1 AND author_id = $2`,
		id, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not unpublish form"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Published form not found"})
		return
	}

	c.Status(http.StatusNoContent)
}
