package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	db "github.com/sidhant-sriv/inventory-api/db"
	"github.com/sidhant-sriv/inventory-api/middleware"
	"github.com/sidhant-sriv/inventory-api/routes"
	"log"
	"os"
)

func main() {
	fmt.Println("Starting Inventory API...")

	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Check if JWT_SECRET_KEY is set
	if os.Getenv("JWT_SECRET_KEY") == "" {
		log.Fatal("JWT_SECRET_KEY environment variable is required")
	}

	// Initialize database
	DB := db.GetDB()
	db.MakeMigration(DB)

	// Set Gin to release mode in production
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Enable detailed SQL logging in development
	// if os.Getenv("GIN_MODE") != "release" {
	//     DB = DB.Debug()
	// }

	// Initialize Gin router with default middleware
	router := gin.Default()

	// Add CORS middleware if needed
	// router.Use(middleware.CORSMiddleware())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// Register routes
	routes.AuthRoutes(router) // Auth routes (public)

	// Protected user routes
	userGroup := router.Group("/users")
	userGroup.POST("/", routes.CreateUser()) // Allow registration without auth

	// Protected routes
	userGroup.Use(middleware.AuthMiddleware())
	{
		userGroup.GET("/:user_id", routes.GetUser())
		userGroup.GET("/", routes.GetAllUsers())
		userGroup.PUT("/:user_id", routes.UpdateUser())
		userGroup.DELETE("/:user_id", routes.DeleteUser())
	}

	// Item routes
	routes.ItemRoutes(router)
	routes.LocationRoutes(router)
	// Get the port from environment variables or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	fmt.Printf("Server running on port %s\n", port)
	router.Run(":" + port)
}
