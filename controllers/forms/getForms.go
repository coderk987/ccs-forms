package controllers

import (
	"ccs-forms/db"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func GetForms(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	rows, err := db.Pool.Query(c.Request.Context(), `
		SELECT id, title, description
		FROM draft_forms
		WHERE author_id = $1
		ORDER BY id DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get draft forms"})
		return
	}
	defer rows.Close()

	forms, err := pgx.CollectRows(rows, pgx.RowToStructByName[DraftForm])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read draft forms"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"forms": forms})
}
