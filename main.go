package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"warehouse/cmd"
	"warehouse/internal/assets"

	"warehouse/internal/database"
	"warehouse/internal/locations"
	"warehouse/internal/repository"
	"warehouse/internal/transfers"
	"warehouse/internal/users"
	"warehouse/pkg/auditlog"
	"warehouse/pkg/security"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func init() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load .env file, but don't overwrite system environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found, falling back to system environment variables.")
	}

	// Execute migration CMD
	cmd.Execute(ctx)
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
		log.Fatalf("Error: %v", err)
	}
	defer db.Close()

	log.Println("Connected to the database successfully!")

	repository := repository.NewRepository(db)
	auditLog := auditlog.NewAuditLog(repository)

	router := gin.Default()
	assets.RegisterRoutes(router, repository, auditLog)
	transfers.RegisterRoutes(router, repository, auditLog)
	locations.RegisterRoutes(router, db, repository)
	security.RegisterRoutes(router, db)
	users.RegisterRoutes(router, db)

	openapiFilePath := "./docs/index.html"
	if _, err := os.Stat(openapiFilePath); err == nil {
		router.GET("/openapi.html", func(c *gin.Context) {
			c.File(openapiFilePath)
		})
		log.Println("Route docs/index.html registered successfully.")
	} else {
		log.Printf("Warning: %s not found. Route /openapi.html will not be registered.\n", openapiFilePath)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		log.Println("Called healthcheck")
	})

	if err := router.Run(os.Getenv("APP_HOST")); err != nil {
		panic(err)
	}
}
