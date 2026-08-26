package routes

import (
	controllers "ccs-forms/controllers/forms"
	"ccs-forms/middleware"

	"github.com/gin-gonic/gin"
)

func FormRoutes(router *gin.RouterGroup) {
	forms := router.Group("/draft_forms")
	forms.Use(middleware.AuthMiddleware())
	forms.GET("/", controllers.GetForms)
	forms.POST("/", controllers.CreateDraftForm)
	forms.GET("/:id", controllers.GetDraftForm)
	forms.PATCH("/:id", controllers.UpdateDraftForm)
	forms.DELETE("/:id", controllers.DeleteDraftForm)
}
