package main

import (
	"log"
	"os"

	"warehouse/internal/albums"
	"warehouse/internal/database"
	"warehouse/internal/items"
	"warehouse/internal/locations"
	"warehouse/internal/users"
	"warehouse/pkg/security"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func init() {
	// Load .env file, but don't overwrite system environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found, falling back to system environment variables.")
	}
}

func main() {
	// Load environment variables
	var err error

	dbURL := os.Getenv("DATABASE_URL")

	if dbURL == "" {
		log.Fatalf("Starting server on port %s", dbURL)
	}

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
	security.RegisterRoutes(router, db)
	users.RegisterRoutes(router, db)

	// Start the HTTP server
	if err := router.Run(os.Getenv("APP_HOST")); err != nil {
		panic(err)
	}
}
