package routes

import (
	controllers "ccs-forms/controllers/responses"
	"ccs-forms/middleware"

	"github.com/gin-gonic/gin"
)

func ResponseRoutes(router *gin.RouterGroup) {
	responses := router.Group("/")
	responses.Use(middleware.AuthMiddleware())

	response := responses.Group("/response")
	response.DELETE("/:id", controllers.DeleteResponse)

	responseInfo := responses.Group("/responses")
	responseInfo.GET("/:id", controllers.GetResponse)
}
