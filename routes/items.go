package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/sidhant-sriv/inventory-api/db"
	"github.com/sidhant-sriv/inventory-api/middleware"
	"github.com/sidhant-sriv/inventory-api/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"net/http"
	"strconv"
	"time"
)

// ItemRoutes sets up the routes for item-related operations
func ItemRoutes(router *gin.Engine) {
	// Public route
	router.POST("/items", CreateItem())

	// Protected routes
	itemRoutes := router.Group("/items")
	itemRoutes.Use(middleware.AuthMiddleware())
	{
		itemRoutes.GET("/:item_id", GetItem())
		itemRoutes.GET("/", GetAllItems())
		itemRoutes.PUT("/:item_id", UpdateItem())
		itemRoutes.DELETE("/:item_id", DeleteItem())
		itemRoutes.GET("/location/:location_id", GetItemByLocation())
		itemRoutes.GET("/user/:user_id", GetItemByUser())
		itemRoutes.GET("/date", GetItemByDate())
		itemRoutes.GET("/date-range", GetItemByDateRange())
		itemRoutes.GET("/page", GetItemByPage())
		itemRoutes.GET("/location/:location_id/date", GetItemByLocationAndDate())

	}
}

// CreateItem handles the creation of a new item
func CreateItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		var item models.Item
		if err := c.ShouldBindJSON(&item); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get the user ID from the JWT token
		userID, _ := c.Get("user_id")

		// Set the UserID field in the item struct
		item.UserID = uint(userID.(float64))

		// Create the item in database
		DB := db.GetDB()
		if result := DB.Create(&item); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create item: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"item": item})
	}
}

// GetItem retrieves an item by ID
func GetItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		itemID := c.Param("item_id")
		var item models.Item

		// Get the item from the database
		DB := db.GetDB()
		if result := DB.Preload(clause.Associations).First(&item, itemID); result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve item: " + result.Error.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"item": item})
	}
}

// GetAllItems retrieves all items for the authenticated user
func GetAllItems() gin.HandlerFunc {
	return func(c *gin.Context) {
		var items []models.Item

		// Get the user ID from the JWT token
		userID, _ := c.Get("user_id")

		// Get all items for the user
		DB := db.GetDB()
		if result := DB.Preload(clause.Associations).Where("user_id = ?", uint(userID.(float64))).Find(&items); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve items: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

// UpdateItem handles the update of an existing item
func UpdateItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		itemID := c.Param("item_id")
		var item models.Item

		// Get the item from the database
		DB := db.GetDB()
		if result := DB.First(&item, itemID); result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve item: " + result.Error.Error()})
			}
			return
		}

		if err := c.ShouldBindJSON(&item); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Update the item in the database
		if result := DB.Save(&item); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update item: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"item": item})
	}
}

// DeleteItem handles the deletion of an item
func DeleteItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		itemID := c.Param("item_id")
		var item models.Item

		// Get the item from the database
		DB := db.GetDB()
		if result := DB.First(&item, itemID); result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve item: " + result.Error.Error()})
			}
			return
		}

		// Delete the item from the database
		if result := DB.Delete(&item); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete item: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Item deleted successfully"})
	}
}

// GetItemByLocation retrieves items by location ID
func GetItemByLocation() gin.HandlerFunc {
	return func(c *gin.Context) {
		locationID := c.Param("location_id")
		var items []models.Item

		// Get all items for the location
		DB := db.GetDB()
		if result := DB.Where("location_id = ?", locationID).Find(&items); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve items: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

// GetItemByUser retrieves items by user ID
func GetItemByUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("user_id")
		var items []models.Item

		// Get all items for the user
		DB := db.GetDB()
		if result := DB.Where("user_id = ?", userID).Find(&items); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve items: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

// GetItemByDate retrieves items by date
func GetItemByDate() gin.HandlerFunc {
	return func(c *gin.Context) {
		date := c.Query("date")
		var items []models.Item

		// Parse the date
		parsedDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
			return
		}

		// Get all items for the date
		DB := db.GetDB()
		if result := DB.Where("DATE(created_at) = ?", parsedDate.Format("2006-01-02")).Find(&items); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve items: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

// GetItemByDateRange retrieves items by date range
func GetItemByDateRange() gin.HandlerFunc {
	return func(c *gin.Context) {
		startDate := c.Query("start_date")
		endDate := c.Query("end_date")
		var items []models.Item

		// Parse the start and end dates
		parsedStartDate, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
			return
		}

		parsedEndDate, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
			return
		}

		// Get all items for the date range
		DB := db.GetDB()
		if result := DB.Where("created_at BETWEEN ? AND ?", parsedStartDate, parsedEndDate).Find(&items); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve items: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

// GetItemByPage retrieves items with pagination
func GetItemByPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.Query("page"))
		pageSize, _ := strconv.Atoi(c.Query("page_size"))

		var items []models.Item
		var total int64

		// Get all items with pagination
		DB := db.GetDB()
		if result := DB.Model(&models.Item{}).Count(&total).Offset((page - 1) * pageSize).Limit(pageSize).Find(&items); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve items: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	}
}

// GetItemByLocationAndDate retrieves items by location ID and date
func GetItemByLocationAndDate() gin.HandlerFunc {
	return func(c *gin.Context) {
		locationID := c.Param("location_id")

		// Parse JSON body to get the timestamp
		var requestBody struct {
			Date int64 `json:"date"` // Unix timestamp
		}

		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
			return
		}

		// Convert Unix timestamp to time.Time
		parsedDate := time.Unix(requestBody.Date, 0)
		var items []models.Item

		// Get all items for the location and date
		DB := db.GetDB()
		if result := DB.Where("location_id = ? AND DATE(created_at) = ?", locationID, parsedDate.Format("2006-01-02")).Find(&items); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve items: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}
