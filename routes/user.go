//routes with all the user related operations using gin
package routes

import (
  "github.com/gin-gonic/gin"
  "github.com/sidhant-sriv/inventory-api/middlewares"
  "github.com/sidhant-sriv/inventory-api/models"
  "github.com/sidhant-sriv/inventory-api/db"
  "net/http"
)

func UserRoutes(incomingRoutes *gin.Engine) {
  incomingRoutes.POST("/user", CreateUser())
  incomingRoutes.GET("/user/:user_id", GetUser())
  incomingRoutes.PUT("/user/:user_id", UpdateUser())
  incomingRoutes.DELETE("/user/:user_id", DeleteUser())
}

