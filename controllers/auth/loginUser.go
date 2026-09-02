package controllers

import (
	"ccs-forms/db"
	"ccs-forms/middleware"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type LoginRequest struct {
	Email string `json:"email"`
}

// LoginUser is the prototype's passwordless email lookup. It deliberately
// shares token issuance with OAuth so all tokens have identical claims and
// signing rules. It is not an identity-proofing mechanism by itself.
func LoginUser(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}

	var userId int
	err := db.Pool.QueryRow(
		c.Request.Context(),
		`SELECT id FROM users WHERE gmail = $1`,
		email,
	).Scan(&userId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "email not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not find user"})
		return
	}

	tokenString, err := middleware.SignToken(int64(userId))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create token"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"token": tokenString,
	})
}
