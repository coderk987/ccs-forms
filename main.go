package main

import (
	"ccs-forms/routes"
	"fmt"

	"github.com/gin-gonic/gin"

	"ccs-forms/db"
)

func main() {
	db.PostgresConnect()
	err := db.SetupDb(db.Pool)
	if err == nil {
		fmt.Println("Setup already done.")
	}

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
