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

type SignupRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// SignupUser creates a local user and returns the same signed token format
// used by Google OAuth. This endpoint is retained for the prototype's
// passwordless flow; production deployments should use Google sign-in or add
// a real email-verification/password mechanism before exposing it publicly.
func SignupUser(c *gin.Context) {
	var req SignupRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	email := strings.TrimSpace(req.Email)
	name := strings.TrimSpace(req.Name)
	if email == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "email and name are required",
		})
		return
	}
	// Email login is intentionally small in this prototype, but normalize the
	// identifier so signup, passwordless login, and Google OAuth use one key.
	email = strings.ToLower(email)

	var userID int64
	err := db.Pool.QueryRow(
		c.Request.Context(),
		`INSERT INTO users (gmail, name)
		 VALUES ($1, $2)
		 ON CONFLICT (gmail) DO NOTHING
		 RETURNING id`,
		email,
		name,
	).Scan(&userID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "email already registered",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create user"})
		return
	}

	tokenString, err := middleware.SignToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token": tokenString,
	})
}
