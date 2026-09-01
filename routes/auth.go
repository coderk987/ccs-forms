package routes

import (
	controllers "ccs-forms/controllers/auth"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	auth.POST("/signup", controllers.SignupUser)
	auth.POST("/login", controllers.LoginUser)
	auth.GET("/google/login", controllers.GoogleLogin)
	auth.GET("/google/callback", controllers.GoogleCallback)
}
