package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DraftForm struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
}

func draftFormID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form id"})
		return 0, false
	}
	return id, true
}
