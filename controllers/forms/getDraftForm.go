package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetDraftForm(c *gin.Context) {
	res, err := createCompleteForm(c)
	if err != nil {
		if writeRequestError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get draft form"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"form": res})
}
