package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func responseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Response id must be a valid integer"})
		return 0, false
	}
	if id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Response id must be greater than zero"})
		return 0, false
	}

	return id, true
}

func publishedFormID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form id must be a valid integer"})
		return 0, false
	}
	if id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form id must be greater than zero"})
		return 0, false
	}

	return id, true
}
