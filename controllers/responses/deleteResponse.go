package controllers

import (
	"ccs-forms/db"
	"net/http"

	"github.com/gin-gonic/gin"
)

func DeleteResponse(c *gin.Context) {
	userID := c.GetInt64("userID")
	id, ok := responseID(c)
	if !ok {
		return
	}

	result, err := db.Pool.Exec(c.Request.Context(),
		`DELETE FROM responses WHERE id=$1 AND author_id=$2`,
		id, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete response"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Response not found"})
		return
	}

	c.JSON(http.StatusNoContent, gin.H{
		"message": "deleted response",
	})
}
