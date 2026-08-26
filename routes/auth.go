package routes

import (
	controllers "ccs-forms/controllers/auth"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")

	auth.POST("/login", controllers.LoginUser)
}
