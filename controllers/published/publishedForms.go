package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type PublishedForm struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Deadline    *time.Time `json:"deadline"`
}

type UpdatePublishedFormRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Deadline    *time.Time `json:"deadline"`
	ViewerIDs   *[]int64   `json:"viewer_ids"`
}

type DraftForm struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
}

type Answer struct {
	QuestionID        int64           `json:"question_id"`
	Payload           json.RawMessage `json:"payload"`
	CheckboxOptionIDs []int64         `json:"checkbox_option_ids"`
}

type FormResponse struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Gmail     string    `json:"gmail"`
	Timestamp time.Time `json:"timestamp"`
	Answers   []Answer  `json:"answers"`
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

func publishedFormID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form id"})
		return 0, false
	}
	return id, true
}
