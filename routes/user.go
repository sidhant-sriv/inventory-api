// routes/user.go
package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/sidhant-sriv/inventory-api/db"
	"github.com/sidhant-sriv/inventory-api/middleware"
	"github.com/sidhant-sriv/inventory-api/models"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"
)

// Example of protecting user routes with authentication
func UserRoutes(router *gin.Engine) {
	// Public route
	router.POST("/users", CreateUser())

	// Protected routes
	userRoutes := router.Group("/users")
	userRoutes.Use(middleware.AuthMiddleware())
	{
		userRoutes.GET("/:user_id", GetUser())
		userRoutes.GET("/", GetAllUsers())
		userRoutes.PUT("/:user_id", UpdateUser())
		userRoutes.DELETE("/:user_id", DeleteUser())
	}
}

// CreateUser handles the creation of a new user
func CreateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		user.Password = string(hashedPassword)

		// Create the user in database
		DB := db.GetDB()
		if result := DB.Create(&user); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user: " + result.Error.Error()})
			return
		}

		// Don't return the password in the response
		user.Password = ""
		c.JSON(http.StatusCreated, gin.H{"user": user})
	}
}

// GetUser retrieves a user by ID
func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.Param("user_id")
		var user models.User

		DB := db.GetDB()
		if result := DB.First(&user, userId); result.Error != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Don't return the password
		user.Password = ""
		c.JSON(http.StatusOK, gin.H{"user": user})
	}
}

// GetAllUsers retrieves all users with pagination
func GetAllUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Pagination parameters
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

		// Calculate offset
		offset := (page - 1) * pageSize

		var users []models.User
		var count int64

		DB := db.GetDB()

		// Get total count
		DB.Model(&models.User{}).Count(&count)

		// Get paginated users
		if result := DB.Limit(pageSize).Offset(offset).Find(&users); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
			return
		}

		// Don't return passwords
		for i := range users {
			users[i].Password = ""
		}

		c.JSON(http.StatusOK, gin.H{
			"users":       users,
			"total":       count,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": (count + int64(pageSize) - 1) / int64(pageSize),
		})
	}
}

// UpdateUser updates a user's information
func UpdateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.Param("user_id")
		var user models.User

		DB := db.GetDB()

		// Check if user exists
		if result := DB.First(&user, userId); result.Error != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Get update data
		var updateData struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password,omitempty"`
		}

		if err := c.ShouldBindJSON(&updateData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Update fields if provided
		if updateData.Name != "" {
			user.Name = updateData.Name
		}

		if updateData.Email != "" {
			// Check if email is already taken
			var existingUser models.User
			if result := DB.Where("email = ? AND id != ?", updateData.Email, userId).First(&existingUser); result.Error == nil {
				c.JSON(http.StatusConflict, gin.H{"error": "Email is already taken"})
				return
			}
			user.Email = updateData.Email
		}

		// Update password if provided
		if updateData.Password != "" {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(updateData.Password), bcrypt.DefaultCost)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
				return
			}
			user.Password = string(hashedPassword)
		}

		// Save updated user
		if result := DB.Save(&user); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
			return
		}

		// Don't return the password
		user.Password = ""
		c.JSON(http.StatusOK, gin.H{"user": user})
	}
}

// DeleteUser deletes a user
func DeleteUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.Param("user_id")
		var user models.User

		DB := db.GetDB()

		// Check if user exists
		if result := DB.First(&user, userId); result.Error != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Delete the user (soft delete with GORM)
		if result := DB.Delete(&user); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User successfully deleted"})
	}
}
