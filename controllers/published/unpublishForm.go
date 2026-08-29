package controllers

import (
	"ccs-forms/db"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func UnpublishForm(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := publishedFormID(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not unpublish form"})
		return
	}
	defer tx.Rollback(ctx)

	var draft DraftForm
	err = tx.QueryRow(ctx, `
		INSERT INTO draft_forms (title, description, author_id)
		SELECT f.title, f.description, f.author_id
		FROM published_forms f
		WHERE f.id = $1 AND f.author_id = $2
		RETURNING id, title, description`, id, userID).Scan(
		&draft.ID, &draft.Title, &draft.Description,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Published form not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not unpublish form"})
		return
	}

	if _, err := tx.Exec(ctx, `
		UPDATE sections SET form_id = $1 WHERE form_id = $2`, draft.ID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not unpublish form"})
		return
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM published_forms WHERE id = $1`, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not unpublish form"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not unpublish form"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"form": draft})
}
