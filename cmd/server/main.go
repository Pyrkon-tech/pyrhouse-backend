package main

import (
	"log"
	"os"

	"warehouse/internal/albums"
	"warehouse/internal/database"
	"warehouse/internal/items"
	"warehouse/internal/locations"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	var err error

	// envPath, _ := filepath.Abs("../../.env")
	// err = godotenv.Load(envPath)
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file: " + err.Error())
	}

	dbURL := os.Getenv("DATABASE_URL")

	// Connect to the database
	db, err := database.NewPostgresConnection(dbURL)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the Gin router
	router := gin.Default()

	// Album routes
	albums.RegisterRoutes(router, db)
	items.RegisterRoutes(router, db)
	locations.RegisterRoutes(router, db)

	// Start the HTTP server
	router.Run("localhost:8080")
}
