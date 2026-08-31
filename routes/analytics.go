package routes

import (
	controllers "ccs-forms/controllers/analytics"
	"ccs-forms/middleware"

	"github.com/gin-gonic/gin"
)

func AnalyticsRoutes(router *gin.RouterGroup) {
	a := router.Group("/published_forms")
	a.Use(middleware.AuthMiddleware())

	a.GET("/:id/analytics", controllers.GetFormAnalytics)
	a.GET("/:id/export", controllers.ExportFormCSV)

}
