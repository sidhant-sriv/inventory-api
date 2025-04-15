package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/sidhant-sriv/inventory-api/db"
	"github.com/sidhant-sriv/inventory-api/middleware"
	"github.com/sidhant-sriv/inventory-api/models"
	"gorm.io/gorm"
	"net/http"
)

// LocationRoutes sets up the routes for location-related operations
func LocationRoutes(router *gin.Engine) {
	// Public route for listing locations
	router.GET("/locations", GetAllLocations())
	
	// Protected routes
	locationRoutes := router.Group("/locations")
	locationRoutes.Use(middleware.AuthMiddleware())
	{
		locationRoutes.POST("/", CreateLocation())
		locationRoutes.GET("/:location_id", GetLocation())
		locationRoutes.PUT("/:location_id", UpdateLocation())
		locationRoutes.DELETE("/:location_id", DeleteLocation())
	}
}

// CreateLocation handles the creation of a new location
func CreateLocation() gin.HandlerFunc {
	return func(c *gin.Context) {
		var location models.Location
		if err := c.ShouldBindJSON(&location); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create the location in database
		DB := db.GetDB()
		if result := DB.Create(&location); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create location: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"location": location})
	}
}

// GetLocation retrieves a location by ID
func GetLocation() gin.HandlerFunc {
	return func(c *gin.Context) {
		locationID := c.Param("location_id")
		var location models.Location

		// Get the location from the database
		DB := db.GetDB()
		if result := DB.First(&location, locationID); result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Location not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve location: " + result.Error.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"location": location})
	}
}

// GetAllLocations retrieves all locations
func GetAllLocations() gin.HandlerFunc {
	return func(c *gin.Context) {
		var locations []models.Location

		// Get all locations
		DB := db.GetDB()
		if result := DB.Find(&locations); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve locations: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"locations": locations})
	}
}

// UpdateLocation handles the update of an existing location
func UpdateLocation() gin.HandlerFunc {
	return func(c *gin.Context) {
		locationID := c.Param("location_id")
		var location models.Location

		// Get the location from the database
		DB := db.GetDB()
		if result := DB.First(&location, locationID); result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Location not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve location: " + result.Error.Error()})
			}
			return
		}

		// Bind the updated location data from the request
		if err := c.ShouldBindJSON(&location); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Update the location in the database
		if result := DB.Save(&location); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update location: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"location": location})
	}
}

// DeleteLocation handles the deletion of a location
func DeleteLocation() gin.HandlerFunc {
	return func(c *gin.Context) {
		locationID := c.Param("location_id")
		var location models.Location

		// Get the location from the database
		DB := db.GetDB()
		if result := DB.First(&location, locationID); result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Location not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve location: " + result.Error.Error()})
			}
			return
		}

		// Check if there are any items linked to this location
		var count int64
		if result := DB.Model(&models.Item{}).Where("location_id = ?", locationID).Count(&count); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check items: " + result.Error.Error()})
			return
		}

		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete location with linked items"})
			return
		}

		// Delete the location from the database
		if result := DB.Delete(&location); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete location: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Location deleted successfully"})
	}
}