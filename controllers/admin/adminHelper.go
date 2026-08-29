package controllers

import (
	"ccs-forms/db"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type GmailRequest struct {
	Gmail string `json:"gmail" binding:"required"`
}

func requireAdmin(c *gin.Context) (int64, bool) {
	userID := c.GetInt64("userID")
	var isAdmin bool
	if err := db.Pool.QueryRow(c.Request.Context(),
		`SELECT role = 'admin' FROM users WHERE id = $1`, userID,
	).Scan(&isAdmin); errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return 0, false
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not verify admin access"})
		return 0, false
	}

	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return 0, false
	}
	return userID, true
}

func parseGmailRequest(c *gin.Context) (string, bool) {
	var req GmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A gmail is required"})
		return "", false
	}

	gmail := strings.TrimSpace(req.Gmail)
	if gmail == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A gmail is required"})
		return "", false
	}
	return gmail, true
}

func adminFormID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form id"})
		return 0, false
	}
	return id, true
}
