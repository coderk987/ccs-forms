package controllers

import (
	"ccs-forms/db"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func AddViewPermission(c *gin.Context) {
	actorID, ok := requireAdmin(c)
	if !ok {
		return
	}
	gmail, ok := parseGmailRequest(c)
	if !ok {
		return
	}

	formID, ok := adminFormID(c)
	if !ok {
		return
	}

	var targetUserID int64
	err := db.Pool.QueryRow(c.Request.Context(),
		`SELECT id FROM users WHERE gmail = $1`, gmail,
	).Scan(&targetUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not find user"})
		return
	}

	var permission struct {
		UserID int64 `json:"user_id"`
		FormID int64 `json:"form_id"`
	}
	err = db.Pool.QueryRow(c.Request.Context(),
		`INSERT INTO view_permissions (user_id, form_id)
		 SELECT $1, f.id
		 FROM published_forms f
		 WHERE f.id = $2 AND f.author_id = $3
		 ON CONFLICT (user_id, form_id) DO UPDATE SET user_id = EXCLUDED.user_id
		 RETURNING user_id, form_id`,
		targetUserID, formID, actorID,
	).Scan(&permission.UserID, &permission.FormID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Published form not found or you are not its author"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not add view permission"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"permission": permission})
}
