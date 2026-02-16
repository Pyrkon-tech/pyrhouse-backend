package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"warehouse/internal/config"
	"warehouse/internal/database"
	di "warehouse/internal/di"
	"warehouse/internal/middleware"
	routes "warehouse/internal/routing"
	"warehouse/internal/security"
)

func main() {
	log.Println("Inicjalizacja aplikacji...")

	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Ostrzeżenie: Nie znaleziono pliku .env: %v", err)
	} else {
		log.Println("Plik .env załadowany pomyślnie")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Validate required config
	if cfg.Database.URL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Parse command line flags
	migrateOnly := flag.Bool("migrate", false, "run only migrations without starting the server")
	migrationsDir := flag.String("dir", "./migrations", "directory containing migration files")
	flag.Parse()

	// Setup DB
	db, err := database.NewPostgresConnection(cfg.Database)
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

	// Initialize security module
	if err := security.Initialize(cfg.JWT); err != nil {
		log.Fatalf("Error initializing security: %v", err)
	}

	// Setup server
	container := di.NewAppContainer(db, cfg)
	defer container.Close() // Ensure cleanup on shutdown
	router := setupRouter(container, cfg)
	middleware.SetVersion(cfg.Server.Version)

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give outstanding requests 10 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

func setupRouter(container *di.Container, cfg *config.Config) *gin.Engine {
	router := gin.Default()

	router.Use(middleware.RecoveryMiddleware())

	if cfg.Server.RequestTimeout > 0 {
		router.Use(middleware.TimeoutMiddleware(cfg.Server.RequestTimeout * time.Second))
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowedOrigins,
		AllowMethods:     cfg.CORS.AllowedMethods,
		AllowHeaders:     cfg.CORS.AllowedHeaders,
		ExposeHeaders:    cfg.CORS.ExposedHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
	}))

	routes.RegisterPublicRoutes(router, container)
	routes.RegisterProtectedRoutes(router, container)
	routes.RegisterUtilityRoutes(router)

	log.Println("[Router]: Setup completed")

	return router
}
