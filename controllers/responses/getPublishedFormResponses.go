package controllers

import (
	"ccs-forms/db"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type FormResponse struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	FormID    int64     `db:"form_id"`
	Timestamp time.Time `db:"timestamp"`
}

func GetPublishedFormResponses(c *gin.Context) {
	form_id, ok := publishedFormID(c)
	if !ok {
		return
	}

	rows, err := db.Pool.Query(c.Request.Context(),
		`SELECT * FROM responses WHERE form_id = $1`,
		form_id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get responses"})
		return
	}
	defer rows.Close()

	responses, err := pgx.CollectRows(rows, pgx.RowToStructByName[FormResponse])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read responses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"responses": responses,
	})
}
