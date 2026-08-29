package routes

import (
	controllers "ccs-forms/controllers/published"
	"ccs-forms/middleware"

	"github.com/gin-gonic/gin"
)

func PublishedFormRoutes(router *gin.RouterGroup) {
	published := router.Group("/published_forms")
	published.Use(middleware.AuthMiddleware())
	published.GET("/", controllers.GetPublishedForms)
	published.PATCH("/:id", controllers.UpdatePublishedForm)
	published.DELETE("/:id", controllers.UnpublishForm)
	published.GET("/:id/responses", controllers.GetPublishedFormResponses)
}
