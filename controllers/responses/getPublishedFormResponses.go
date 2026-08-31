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
	userID := c.GetInt64("userID")
	form_id, ok := publishedFormID(c)
	if !ok {
		return
	}

	rows, err := db.Pool.Query(c.Request.Context(),
		`SELECT r.id, r.user_id, r.form_id, r."timestamp"
		 FROM responses r
		 JOIN published_forms f ON f.id = r.form_id
		 WHERE r.form_id = $1
		   AND (f.author_id = $2 OR EXISTS (
			 SELECT 1 FROM view_permissions v
			 WHERE v.form_id = f.id AND v.user_id = $2
		   ))
		 ORDER BY r."timestamp" DESC`,
		form_id, userID,
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
