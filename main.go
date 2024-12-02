package main

import (
	"context"
	"log"
	"os"
	"time"

	"warehouse/cmd"
	"warehouse/internal/container"
	"warehouse/internal/routes"

	"warehouse/internal/database"

	"github.com/gin-contrib/cors"
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

	cmd.Execute(ctx)
}

func main() {
	var err error

	// Setup DB
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatalf("Starting server on port %s", dbURL)
	}
	db, err := database.NewPostgresConnection(dbURL)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer db.Close()
	log.Println("[DB]: Setup completed")

	container := container.NewAppContainer(db)
	router := setupRouter(container)

	if err := router.Run(os.Getenv("APP_HOST")); err != nil {
		panic(err)
	}
}

func setupRouter(container *container.Container) *gin.Engine {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5000"}, // Add your domain and localhost
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	routes.RegisterPublicRoutes(router, container)
	routes.RegisterProtectedRoutes(router, container)
	routes.RegisterUtilityRoutes(router)

	log.Println("[Router]: Setup completed")

	return router
}
