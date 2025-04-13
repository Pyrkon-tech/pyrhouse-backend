package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"warehouse/internal/core/container"
	"warehouse/internal/core/routes"
	"warehouse/internal/database"
	"warehouse/internal/middleware"
)

func init() {
	// Load .env file, but don't overwrite system environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found, falling back to system environment variables.")
	}
}

func main() {
	var err error

	// Parse command line flags
	migrateOnly := flag.Bool("migrate", false, "run only migrations without starting the server")
	migrationsDir := flag.String("dir", "./migrations", "directory containing migration files")
	flag.Parse()

	// Setup DB
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}
	db, err := database.NewPostgresConnection(dbURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()
	log.Println("[DB]: Setup completed")

	// Run migrations if requested
	if *migrateOnly {
		if err := database.RunMigrations(db, *migrationsDir); err != nil {
			log.Fatalf("Error running migrations: %v", err)
		}
		log.Println("[Migrations]: Completed successfully")
		return
	}

	// Start server
	container := container.NewAppContainer(db)
	router := setupRouter(container)

	// Ustawienie wersji aplikacji
	middleware.SetVersion("1.0.0")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRouter(container *container.Container) *gin.Engine {
	router := gin.Default()

	// Dodanie middleware do odzyskiwania po awariach
	router.Use(middleware.RecoveryMiddleware())

	// Timeout tylko jeśli REQUEST_TIMEOUT jest ustawiony
	timeoutStr := os.Getenv("REQUEST_TIMEOUT")
	if timeoutStr != "" {
		if timeoutSeconds, err := strconv.Atoi(timeoutStr); err == nil && timeoutSeconds > 0 {
			timeout := time.Duration(timeoutSeconds) * time.Second
			router.Use(middleware.TimeoutMiddleware(timeout))
		}
	}

	// Endpoint /health jest już zarejestrowany w routes.RegisterUtilityRoutes

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5000", "https://pyrhouse-frontend-p2sbw.ondigitalocean.app"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
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
