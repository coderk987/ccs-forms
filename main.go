package main

import (
	"ccs-forms/middleware"
	"ccs-forms/routes"
	"fmt"
	"os"

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
	router.Use(middleware.CORSMiddleware())
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Running Server",
		})
	})
	api := router.Group("/")
	routes.FormRoutes(api)
	routes.ResponderRoutes(api)
	routes.ResponseRoutes(api)
	routes.AdminRoutes(api)
	routes.AuthRoutes(api)
	routes.AnalyticsRoutes(api)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := router.Run("0.0.0.0:" + port); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
