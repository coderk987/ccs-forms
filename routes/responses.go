package routes

import (
	controllers "ccs-forms/controllers/responses"

	"github.com/gin-gonic/gin"
)

func ResponseRoutes(router *gin.RouterGroup) {
	responses := router.Group("/responses")
	responses.GET("/", controllers.GetResponse)
}
