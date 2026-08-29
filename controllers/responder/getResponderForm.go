package controllers

import (
	"ccs-forms/db"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func draftFormID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form id must be a valid integer"})
		return 0, false
	}
	if id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form id must be greater than zero"})
		return 0, false
	}
	return id, true
}

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
