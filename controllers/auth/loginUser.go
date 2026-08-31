package controllers

import (
	"ccs-forms/db"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/golang-jwt/jwt/v5"

	"os"
	"time"

	"errors"
	"strconv"
)

type LoginRequest struct {
	Email string `json:"email"`
}

func LoginUser(c *gin.Context) {
	var req LoginRequest

	fmt.Println("Running Login")

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	var userId int
	err := db.Pool.QueryRow(
		c.Request.Context(),
		`SELECT id FROM users WHERE gmail = $1`,
		req.Email,
	).Scan(&userId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(404, gin.H{
				"error":   "email not found",
				"message": err.Error(),
			})
			return
		}
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	claims := jwt.RegisteredClaims{
		Subject:   strconv.Itoa(userId),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create token"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"token": tokenString,
	})
}
