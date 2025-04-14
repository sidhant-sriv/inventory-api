package db

import (
  "fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
  "github.com/joho/godotenv"
  "github.com/sidhant-sriv/inventory-api/models"
  "log"
  "os"
)

var DB *gorm.DB

func DbConnect() {
  err := godotenv.Load()
  if err != nil {
    log.Fatal("Error loading .env file")
  }

  dsn := fmt.Sprintf(
    "host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Kolkata",
    os.Getenv("DB_HOST"),
    os.Getenv("DB_USERNAME"),
    os.Getenv("DB_PASSWORD"),
    os.Getenv("DB_NAME"),
    os.Getenv("DB_PORT"),
    )
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
      log.Fatal("Failed to connect to the database")
  }
  DB = db 
  fmt.Println("Connected to the database")
}

func GetDB() *gorm.DB {
  if DB == nil {
    DbConnect()
  }
  return DB
}

func MakeMigration(DB *gorm.DB) {
  DB.AutoMigrate(&models.User{})
  fmt.Println("Database migrated successfully")
}
