// routes/auth.go
package routes

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/sidhant-sriv/inventory-api/db"
	"github.com/sidhant-sriv/inventory-api/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm" // Import gorm if you need to check for specific gorm errors like ErrRecordNotFound
)

// AuthRoutes sets up the authentication routes /auth/register, /auth/login, etc.
func AuthRoutes(router *gin.Engine) {
	auth := router.Group("/auth")
	{
		auth.POST("/register", Register())
		auth.POST("/login", Login())
		auth.POST("/refresh", RefreshToken())
		auth.GET("/check-user", CheckUserExists()) // Debug endpoint
	}
}

// Register handles new user registration.
func Register() gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User
		var registerRequest struct {
			Name     string `json:"name" binding:"required"`
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required,min=6"` // Example: minimum 6 characters
		}

		// Basic validation (consider adding more robust validation)
		if err := c.ShouldBindJSON(&registerRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
			return
		}

		// Map the validated request to the User model
		user.Name = registerRequest.Name
		user.Email = registerRequest.Email
		user.Password = registerRequest.Password

		// Hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			fmt.Printf("Error hashing password: %v\n", err) // Log internal error
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process registration"})
			return
		}
		// Store the hashed password, not the plain text one
		user.Password = string(hashedPassword)

		// Get DB connection
		DB := db.GetDB()
		if DB == nil {
			fmt.Println("Error: Database connection is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
			return
		}

		// Create the user in database
		// Clear the plain text password before saving (though it's already hashed)
		// user.Password = string(hashedPassword) // Already done above

		if result := DB.Create(&user); result.Error != nil {
			// Check for duplicate email or other DB constraints
			// Note: Specific error checking might depend on your database driver
			fmt.Printf("Error creating user in DB: %v\n", result.Error) // Log internal error
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user. Email might already be registered."})
			return
		}

		// Generate JWT tokens
		accessToken, refreshToken, err := generateTokens(user.ID)
		if err != nil {
			fmt.Printf("Error generating tokens: %v\n", err) // Log internal error
			// Consider if user should be informed or if this requires cleanup
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize registration"})
			return
		}

		// Return newly created user info (excluding password) and tokens
		c.JSON(http.StatusCreated, gin.H{
			"message": "User registered successfully",
			"user": gin.H{
				"id":    user.ID,
				"name":  user.Name,
				"email": user.Email,
			},
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}

// Login handles user login requests.
func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		var loginRequest struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}

		// Bind JSON payload
		if err := c.ShouldBindJSON(&loginRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
			return
		}

		// Get DB connection
		DB := db.GetDB()
		if DB == nil {
			fmt.Println("Error: Database connection is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
			return
		}

		// Find user by email
		var user models.User
		fmt.Printf("Attempting login with email: %s\n", loginRequest.Email) // Debug log

		// Use First to get a single record. It returns gorm.ErrRecordNotFound if no user is found.
		result := DB.Where("email = ?", loginRequest.Email).First(&user)

		// Check if user was found
		if result.Error != nil {
			fmt.Printf("Database error during login lookup for email %s: %v\n", loginRequest.Email, result.Error) // Log internal error
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"}) // User not found
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error during login"}) // Other DB error
			}
			return
		}

		fmt.Printf("Found user with ID: %d. Comparing password.\n", user.ID) // Debug log
		// fmt.Printf("Stored Hash: %s\n", user.Password) // Optional: Debug log for hash (be careful in prod)
		// fmt.Printf("Password Attempt: %s\n", loginRequest.Password) // Don't log plaintext passwords in production

		// Verify password using bcrypt
		err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginRequest.Password))
		if err != nil {
			// Password does not match
			fmt.Printf("Password comparison failed for user ID %d: %v\n", user.ID, err) // Log internal error (usually bcrypt.ErrMismatchedHashAndPassword)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		// Password is correct, generate tokens
		accessToken, refreshToken, err := generateTokens(user.ID)
		if err != nil {
			fmt.Printf("Error generating tokens for user ID %d: %v\n", user.ID, err) // Log internal error
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate login tokens"})
			return
		}

		// Return user info (excluding password) and tokens
		c.JSON(http.StatusOK, gin.H{
			"message": "Login successful",
			"user": gin.H{
				"id":    user.ID,
				"name":  user.Name,
				"email": user.Email,
			},
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}

// RefreshToken handles requests to refresh JWT access tokens using a valid refresh token.
func RefreshToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		var refreshRequest struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}

		// Bind JSON payload
		if err := c.ShouldBindJSON(&refreshRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
			return
		}

		// Get JWT secret from environment
		jwtSecret := os.Getenv("JWT_SECRET_KEY")
		if jwtSecret == "" {
			fmt.Println("Error: JWT_SECRET_KEY environment variable not set.")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Server configuration error"})
			return
		}

		// Parse the refresh token
		token, err := jwt.Parse(refreshRequest.RefreshToken, func(token *jwt.Token) (interface{}, error) {
			// Validate the algorithm (HS256 in this case)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		// Check for parsing errors or invalid token
		if err != nil || !token.Valid {
			fmt.Printf("Invalid refresh token received: %v\n", err) // Log internal error/reason
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
			return
		}

		// Extract claims from the token
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			fmt.Println("Error: Failed to parse token claims")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not process token"})
			return
		}

		// Check if it's actually a refresh token (based on the 'type' claim)
		if tokenType, ok := claims["type"].(string); !ok || tokenType != "refresh" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token type provided"})
			return
		}

		// Extract user ID from claims
		userIDFloat, ok := claims["user_id"].(float64) // JWT numbers are often float64
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not parse user ID from token"})
			return
		}
		userID := uint(userIDFloat)

		// Optional: Verify user still exists in the database
		DB := db.GetDB()
		if DB == nil {
			fmt.Println("Error: Database connection is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
			return
		}

		var user models.User
		if result := DB.First(&user, userID); result.Error != nil {
			fmt.Printf("User ID %d from refresh token not found in DB: %v\n", userID, result.Error)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User associated with token not found"})
			return
		}

		// Generate new access and refresh tokens
		newAccessToken, newRefreshToken, err := generateTokens(userID)
		if err != nil {
			fmt.Printf("Error generating tokens during refresh for user ID %d: %v\n", userID, err) // Log internal error
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate new tokens"})
			return
		}

		// Return the new tokens
		c.JSON(http.StatusOK, gin.H{
			"message":       "Tokens refreshed successfully",
			"access_token":  newAccessToken,
			"refresh_token": newRefreshToken, // Return a new refresh token as well for sliding sessions
		})
	}
}

// CheckUserExists is a debug endpoint to verify if a user exists by email.
func CheckUserExists() gin.HandlerFunc {
	return func(c *gin.Context) {
		email := c.Query("email")
		if email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email query parameter is required"})
			return
		}

		DB := db.GetDB()
		if DB == nil {
			fmt.Println("Error: Database connection is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
			return
		}

		var user models.User
		result := DB.Where("email = ?", email).Select("id", "name", "email").First(&user) // Select only needed fields

		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusOK, gin.H{"exists": false, "message": "User not found"})
			} else {
				fmt.Printf("Database error checking user existence for email %s: %v\n", email, result.Error)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "details": result.Error.Error()})
			}
			return
		}

		// User exists, return selected details (no password)
		c.JSON(http.StatusOK, gin.H{
			"exists":  true,
			"user_id": user.ID,
			"name":    user.Name,
			"email":   user.Email,
		})
	}
}

// generateTokens is a helper function to create new JWT access and refresh tokens.
func generateTokens(userID uint) (string, string, error) {
	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		fmt.Println("CRITICAL: JWT_SECRET_KEY environment variable not set.")
		return "", "", fmt.Errorf("JWT secret key not configured")
	}
	secretKeyBytes := []byte(jwtSecret)

	// Create access token (shorter lifespan)
	accessTokenClaims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 1).Unix(), // Expires in 1 hour
		"iat":     time.Now().Unix(),                    // Issued at
		"type":    "access",                             // Token type identifier
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)
	accessTokenString, err := accessToken.SignedString(secretKeyBytes)
	if err != nil {
		fmt.Printf("Error signing access token: %v\n", err)
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	// Create refresh token (longer lifespan)
	refreshTokenClaims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // Expires in 7 days
		"iat":     time.Now().Unix(),                         // Issued at
		"type":    "refresh",                                 // Token type identifier
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims)
	refreshTokenString, err := refreshToken.SignedString(secretKeyBytes)
	if err != nil {
		fmt.Printf("Error signing refresh token: %v\n", err)
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return accessTokenString, refreshTokenString, nil
}
