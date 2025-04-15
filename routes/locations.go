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
	router.GET("/locations/public", GetPublicLocations())

	// Protected routes
	locationRoutes := router.Group("/locations")
	locationRoutes.Use(middleware.AuthMiddleware())
	{
		locationRoutes.POST("/", CreateLocation())
		locationRoutes.GET("/", GetUserLocations())
		locationRoutes.GET("/:location_id", GetLocation())
		locationRoutes.PUT("/:location_id", UpdateLocation())
		locationRoutes.DELETE("/:location_id", DeleteLocation())
	}
}

// GetPublicLocations retrieves all public locations (not associated with specific users)
func GetPublicLocations() gin.HandlerFunc {
	return func(c *gin.Context) {
		var locations []models.Location

		// Get public locations (where UserID is 0 or NULL)
		DB := db.GetDB()
		if result := DB.Where("user_id = 0 OR user_id IS NULL").Find(&locations); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve public locations: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"locations": locations})
	}
}

// GetUserLocations retrieves all locations for the authenticated user
func GetUserLocations() gin.HandlerFunc {
	return func(c *gin.Context) {
		var locations []models.Location

		// Get the user ID from the JWT token
		userID := middleware.GetUserID(c)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}

		// Get all locations for the user (include public locations too)
		DB := db.GetDB()
		if result := DB.Where("user_id = ? OR user_id = 0 OR user_id IS NULL", userID).Find(&locations); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve locations: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"locations": locations})
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

		// Set the UserID from the authenticated user
		userID := middleware.GetUserID(c)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}
		location.UserID = userID

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

		// Get the user ID from the JWT token
		userID := middleware.GetUserID(c)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}

		// Get the location from the database
		DB := db.GetDB()
		// Allow access if location is public (user_id = 0 or NULL) or owned by the current user
		if result := DB.Where("id = ? AND (user_id = ? OR user_id = 0 OR user_id IS NULL)", locationID, userID).First(&location); result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Location not found or access denied"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve location: " + result.Error.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"location": location})
	}
}

// UpdateLocation handles the update of an existing location
func UpdateLocation() gin.HandlerFunc {
	return func(c *gin.Context) {
		locationID := c.Param("location_id")
		var location models.Location

		// Get the user ID from the JWT token
		userID := middleware.GetUserID(c)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}

		// Get the location from the database
		DB := db.GetDB()
		if result := DB.Where("id = ? AND user_id = ?", locationID, userID).First(&location); result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Location not found or you don't have permission to update it"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve location: " + result.Error.Error()})
			}
			return
		}

		// Bind the updated location data from the request
		var updateData models.Location
		if err := c.ShouldBindJSON(&updateData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Update fields (preserving UserID)
		location.Name = updateData.Name
		location.Description = updateData.Description
		location.ImageUrl = updateData.ImageUrl
		// Don't allow changing the UserID

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

		// Get the user ID from the JWT token
		userID := middleware.GetUserID(c)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}

		// Get the location from the database with user ownership check
		DB := db.GetDB()
		var location models.Location
		if result := DB.Where("id = ? AND user_id = ?", locationID, userID).First(&location); result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Location not found or you don't have permission to delete it"})
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
