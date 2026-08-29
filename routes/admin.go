package routes

import (
	controllers "ccs-forms/controllers/admin"
	"ccs-forms/middleware"

	"github.com/gin-gonic/gin"
)

func AdminRoutes(router *gin.RouterGroup) {
	admin := router.Group("/admin")
	admin.Use(middleware.AuthMiddleware())

	admin.PATCH("/users/admin", controllers.MakeAdmin)
	admin.POST("/forms/:id/view-permissions", controllers.AddViewPermission)
}
