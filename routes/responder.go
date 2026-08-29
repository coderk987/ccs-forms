package routes

import (
	controllers "ccs-forms/controllers/responder"
	"ccs-forms/middleware"

	"github.com/gin-gonic/gin"
)

func ResponderRoutes(router *gin.RouterGroup) {
	form := router.Group("/forms")
	form.Use(middleware.AuthMiddleware())

	form.GET("/:id", controllers.GetResponderForm)
	form.POST("/:id/response", controllers.CreateResponse)
}
