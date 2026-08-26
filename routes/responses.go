package routes

import (
	controllers "ccs-forms/controllers/responses"
	"ccs-forms/middleware"

	"github.com/gin-gonic/gin"
)

func ResponseRoutes(router *gin.RouterGroup) {
	responses := router.Group("/responses")
	responses.Use(middleware.AuthMiddleware())
	responses.GET("/", controllers.GetResponse)
}
