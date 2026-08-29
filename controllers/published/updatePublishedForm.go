package controllers

import (
	"ccs-forms/db"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func UpdatePublishedForm(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := publishedFormID(c)
	if !ok {
		return
	}

	var req UpdatePublishedFormRequest
	if err := c.ShouldBindJSON(&req); err != nil ||
		(req.Title == nil && req.Description == nil && req.Deadline == nil && req.ViewerIDs == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide a title, description, deadline or viewers to update"})
		return
	}

	ctx := c.Request.Context()

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update published form"})
		return
	}
	defer tx.Rollback(ctx)

	var form PublishedForm
	err = tx.QueryRow(ctx, `
		UPDATE published_forms
		SET title = COALESCE($1, title),
			description = COALESCE($2, description),
			deadline = COALESCE($3, deadline)
		WHERE id = $4 AND author_id = $5
		RETURNING id, title, description, deadline`,
		req.Title, req.Description, req.Deadline, id, userID).Scan(
		&form.ID, &form.Title, &form.Description, &form.Deadline,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Published form not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update published form"})
		return
	}

	if req.ViewerIDs != nil {
		if _, err := tx.Exec(ctx, `
			DELETE FROM view_permissions
			WHERE form_id = $1 AND user_id <> ALL($2)`, id, *req.ViewerIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update viewers"})
			return
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO view_permissions (user_id, form_id)
			SELECT unnest($1::bigint[]), $2
			ON CONFLICT DO NOTHING`, *req.ViewerIDs, id); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown viewer"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update viewers"})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update published form"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"form": form})
}
