package main

import (
	"ccs-forms/routes"

	"github.com/gin-gonic/gin"
)

func main() {

	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Running Server",
		})
	})
	api := router.Group("/")
	routes.FormRoutes(api)
	routes.ResponseRoutes(api)
	router.Run(":3000")
}
