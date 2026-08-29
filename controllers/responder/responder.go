package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type Response struct {
	ID        int64     `json:"id"`
	FormID    int64     `json:"form_id"`
	Timestamp time.Time `json:"timestamp"`
}

type AnswerRequest struct {
	QuestionID int64           `json:"question_id"`
	Payload    json.RawMessage `json:"payload"`
}

type CreateResponseRequest struct {
	Answers map[string]json.RawMessage `json:"answers" binding:"required"`
}

func currentUserID(c *gin.Context) (int64, bool) {
	claim, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return 0, false
	}

	switch value := claim.(type) {
	case int64:
		return value, true
	case int:
		return int64(value), true
	case string:
		userID, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
			return 0, false
		}
		return userID, true
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return 0, false
	}
}

func publishedFormID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form id"})
		return 0, false
	}
	return id, true
}

type FormOption struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	UnlockSectionID *int64 `json:"unlock_section_id,omitempty"`
}

type FormQuestion struct {
	ID         int64           `json:"id"`
	Title      string          `json:"title"`
	Type       string          `json:"type"`
	Validation json.RawMessage `json:"validation"`
	Options    []FormOption    `json:"options"`
}

type FormSection struct {
	ID          int64          `json:"id"`
	Title       string         `json:"title"`
	Description *string        `json:"description"`
	Questions   []FormQuestion `json:"questions"`
}

type ResponderForm struct {
	ID          int64         `json:"id"`
	Title       string        `json:"title"`
	Description *string       `json:"description"`
	Deadline    *time.Time    `json:"deadline"`
	Closed      bool          `json:"closed"`
	Sections    []FormSection `json:"sections"`
}
