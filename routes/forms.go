package routes

import (
	controllers "ccs-forms/controllers/forms"
	"ccs-forms/middleware"

	"github.com/gin-gonic/gin"
)

func FormRoutes(router *gin.RouterGroup) {
	forms := router.Group("/forms")
	forms.Use(middleware.AuthMiddleware())
	forms.GET("/", controllers.GetForms)
}
