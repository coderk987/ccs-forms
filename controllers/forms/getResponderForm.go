package controllers

import (
	"ccs-forms/db"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetResponderForm(c *gin.Context) {
	id, ok := draftFormID(c)
	if !ok {
		return
	}
	userID := c.GetInt64("userID")

	var row json.RawMessage
	err := db.Pool.QueryRow(c.Request.Context(),
		`SELECT structure FROM published_forms WHERE id=$1 AND author_id = $2 ORDER BY id DESC`,
		id, userID,
	).Scan(&row)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get published forms"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"form": row,
	})
}
