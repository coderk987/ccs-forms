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

type CreateDraftFormRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description *string `json:"description"`
}

type UpdateDraftFormRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

func currentUserID(c *gin.Context) (int64, bool) {
	claim, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return 0, false
	}

	userIDClaim, ok := claim.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return 0, false
	}

	userID, err := strconv.ParseInt(userIDClaim, 10, 64)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return 0, false
	}

	return userID, true
}

func draftFormID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form id"})
		return 0, false
	}
	return id, true
}
