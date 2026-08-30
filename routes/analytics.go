package routes

import (
	"ccs-forms/analytics"
	"ccs-forms/middleware"

	"github.com/gin-gonic/gin"
)

func AnalyticsRoutes(router *gin.RouterGroup) {
	a := router.Group("/published_forms")
	a.Use(middleware.AuthMiddleware())

	a.GET("/:id/analytics", analytics.GetFormAnalytics)
	a.GET("/:id/export", analytics.ExportFormCSV)

}
