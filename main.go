package main

import (
  "fmt"
  db "github.com/sidhant-sriv/inventory-api/db"

) 


func main() {
  fmt.Println("Hello, World!")
  DB := db.GetDB()
  db.MakeMigration(DB)
}
