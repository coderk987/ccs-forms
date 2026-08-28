package controllers

import (
	"ccs-forms/db"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func GetResponderForm(c *gin.Context) {
	id, ok := draftFormID(c)
	if !ok {
		return
	}

	var row json.RawMessage
	err := db.Pool.QueryRow(c.Request.Context(),
		`SELECT structure FROM published_forms WHERE id=$1 ORDER BY id DESC`,
		id,
	).Scan(&row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Published form not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get published forms"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"form": row,
	})
}
