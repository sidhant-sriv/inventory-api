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
	// All item routes should be protected
	itemRoutes := router.Group("/items")
	itemRoutes.Use(middleware.AuthMiddleware())
	{
		itemRoutes.POST("/", CreateItem())
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
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}

		// Set the UserID field in the item struct with proper type checking
		id, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
			return
		}
		item.UserID = id

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

		// Verify user owns this item
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}

		id, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
			return
		}

		if item.UserID != id {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to view this item"})
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
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}

		id, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
			return
		}

		// Get all items for the user
		DB := db.GetDB()
		if result := DB.Preload(clause.Associations).Where("user_id = ?", id).Find(&items); result.Error != nil {
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

		// Get the authenticated user ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}

		id, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
			return
		}

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

		// Verify user owns this item
		if item.UserID != id {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to update this item"})
			return
		}

		// Store the current UserID before binding JSON
		originalUserID := item.UserID

		if err := c.ShouldBindJSON(&item); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Prevent changing the user ID
		item.UserID = originalUserID

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

		// Get the authenticated user ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}

		id, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
			return
		}

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

		// Verify user owns this item
		if item.UserID != id {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to delete this item"})
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

		// Get the authenticated user ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}

		id, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
			return
		}

		// Get all items for the location AND the authenticated user
		DB := db.GetDB()
		if result := DB.Where("location_id = ? AND user_id = ?", locationID, id).Find(&items); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve items: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

// GetItemByUser retrieves items by user ID (only if requesting own items)
func GetItemByUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestedUserID := c.Param("user_id")
		var items []models.Item

		// Get the authenticated user ID
		authUserID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}

		id, ok := authUserID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
			return
		}

		// Convert requested user ID to uint for comparison
		reqID, err := strconv.ParseUint(requestedUserID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
			return
		}

		// Only allow users to get their own items
		if uint(reqID) != id {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only view your own items"})
			return
		}

		// Get all items for the user
		DB := db.GetDB()
		if result := DB.Where("user_id = ?", requestedUserID).Find(&items); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve items: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

// GetItemByDate retrieves items by date (only for the authenticated user)
func GetItemByDate() gin.HandlerFunc {
	return func(c *gin.Context) {
		date := c.Query("date")
		if date == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Date parameter is required"})
			return
		}

		// Get the authenticated user ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}

		id, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
			return
		}

		var items []models.Item

		// Parse the date
		parsedDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
			return
		}

		// Get all items for the date AND the authenticated user
		DB := db.GetDB()
		if result := DB.Where("DATE(created_at) = ? AND user_id = ?", parsedDate.Format("2006-01-02"), id).Find(&items); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve items: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

// GetItemByDateRange retrieves items by date range (only for the authenticated user)
func GetItemByDateRange() gin.HandlerFunc {
	return func(c *gin.Context) {
		startDate := c.Query("start_date")
		endDate := c.Query("end_date")

		if startDate == "" || endDate == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Both start_date and end_date parameters are required"})
			return
		}

		// Get the authenticated user ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}

		id, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
			return
		}

		var items []models.Item

		// Parse the start and end dates
		parsedStartDate, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format. Use YYYY-MM-DD"})
			return
		}

		parsedEndDate, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format. Use YYYY-MM-DD"})
			return
		}

		// Get all items for the date range AND the authenticated user
		DB := db.GetDB()
		if result := DB.Where("created_at BETWEEN ? AND ? AND user_id = ?", parsedStartDate, parsedEndDate, id).Find(&items); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve items: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

// GetItemByPage retrieves items with pagination (only for the authenticated user)
func GetItemByPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get pagination parameters with defaults
		pageStr := c.DefaultQuery("page", "1")
		pageSizeStr := c.DefaultQuery("page_size", "10")

		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page parameter"})
			return
		}

		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 || pageSize > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page_size parameter (must be 1-100)"})
			return
		}

		// Get the authenticated user ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}

		id, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
			return
		}

		var items []models.Item
		var total int64

		// Get all items with pagination for the authenticated user
		DB := db.GetDB()
		if result := DB.Model(&models.Item{}).Where("user_id = ?", id).Count(&total); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count items: " + result.Error.Error()})
			return
		}

		if result := DB.Where("user_id = ?", id).Offset((page - 1) * pageSize).Limit(pageSize).Find(&items); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve items: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"items":       items,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		})
	}
}

// GetItemByLocationAndDate retrieves items by location ID and date (changed to use query params)
func GetItemByLocationAndDate() gin.HandlerFunc {
	return func(c *gin.Context) {
		locationID := c.Param("location_id")
		date := c.Query("date")

		if date == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Date query parameter is required"})
			return
		}

		// Get the authenticated user ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}

		id, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
			return
		}

		// Parse the date (accept both YYYY-MM-DD format and Unix timestamp)
		var parsedDate time.Time
		var err error

		// Try first as YYYY-MM-DD
		parsedDate, err = time.Parse("2006-01-02", date)
		if err != nil {
			// Try as Unix timestamp
			timestamp, err := strconv.ParseInt(date, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD or Unix timestamp"})
				return
			}
			parsedDate = time.Unix(timestamp, 0)
		}

		var items []models.Item

		// Get all items for the location, date AND the authenticated user
		DB := db.GetDB()
		if result := DB.Where("location_id = ? AND DATE(created_at) = ? AND user_id = ?",
			locationID, parsedDate.Format("2006-01-02"), id).Find(&items); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve items: " + result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}
