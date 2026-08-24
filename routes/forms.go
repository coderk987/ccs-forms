package routes

import (
	controllers "ccs-forms/controllers/forms"

	"github.com/gin-gonic/gin"
)

func FormRoutes(router *gin.RouterGroup) {
	forms := router.Group("/forms")

	forms.GET("/", controllers.GetForms)
}
