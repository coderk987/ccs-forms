package controllers

import (
	"ccs-forms/db"
	"net/http"
	"time"

	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type PublishedForm struct {
	ID          int64           `db:"id"`
	Title       string          `db:"title"`
	Description string          `db:"description"`
	AuthorID    int64           `db:"author_id"`
	Deadline    time.Time       `db:"deadline"`
	Structure   json.RawMessage `db:"structure"`
}

func GetPublishedForms(c *gin.Context) {
	userID := c.GetInt64("userID")

	rows, err := db.Pool.Query(c.Request.Context(),
		`SELECT * FROM published_forms WHERE author_id = $1 ORDER BY id DESC`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get draft forms"})
		return
	}
	defer rows.Close()

	forms, err := pgx.CollectRows(rows, pgx.RowToStructByName[PublishedForm])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read draft forms"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"forms": forms})
}
