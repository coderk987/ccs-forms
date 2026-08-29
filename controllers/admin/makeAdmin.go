package controllers

import (
	"ccs-forms/db"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func MakeAdmin(c *gin.Context) {
	actorID, ok := requireAdmin(c)
	if !ok {
		return
	}
	gmail, ok := parseGmailRequest(c)
	if !ok {
		return
	}

	var userID int64
	err := db.Pool.QueryRow(c.Request.Context(),
		`UPDATE users
		 SET role = 'admin'
		 WHERE gmail = $1 AND id <> $2
		 RETURNING id`,
		gmail, actorID,
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found or already the current user"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not make user an admin"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user_id": userID, "gmail": gmail, "role": "admin"})
}
