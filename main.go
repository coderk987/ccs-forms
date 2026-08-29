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
	if err != nil {
		fmt.Println("Setup already done.")
		fmt.Println(err.Error())
	}

	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Running Server",
		})
	})
	api := router.Group("/")
	routes.FormRoutes(api)
	routes.ResponderRoutes(api)
	routes.ResponseRoutes(api)
	routes.AuthRoutes(api)

	router.Run(":8080")
}
