package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func UnpublishForm(c *gin.Context) {
	id, ok := draftFormID(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "unpublish form controller wired",
		"form_id": id,
	})
}
