package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

	// Parse command line flags
	migrateOnly := flag.Bool("migrate", false, "run only migrations without starting the server")
	migrationsDir := flag.String("dir", "./migrations", "directory containing migration files")
	flag.Parse()

	middleware.SetVersion(cfg.Server.Version)

	// Without DATABASE_URL the app cannot serve any real traffic, but it must still
	// boot and answer the platform health check instead of crash-looping.
	// Set DATABASE_URL and redeploy to start in full mode.
	if cfg.Database.URL == "" {
		if *migrateOnly {
			log.Fatal("DATABASE_URL environment variable is not set - cannot run migrations")
		}
		log.Println("[WARN] DATABASE_URL is not set - starting in degraded mode (only /health is served)")
		runServer(setupDegradedRouter(cfg, "DATABASE_URL is not configured"), cfg)
		return
	}

	// Setup DB
	db, err := database.NewPostgresConnection(cfg.Database)
	if err != nil {
		if *migrateOnly {
			log.Fatalf("Error connecting to database: %v", err)
		}
		log.Printf("[WARN] Error connecting to database: %v - starting in degraded mode (only /health is served)", err)
		runServer(setupDegradedRouter(cfg, "database connection is unavailable"), cfg)
		return
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
	if cfg.JWT.Secret == "" {
		secret, genErr := generateEphemeralSecret()
		if genErr != nil {
			log.Fatalf("Error generating ephemeral JWT secret: %v", genErr)
		}
		cfg.JWT.Secret = secret
		log.Println("[WARN] JWT_SECRET is not set - using an ephemeral secret; issued tokens break on restart and are not shared between instances")
	}
	if err := security.Initialize(cfg.JWT); err != nil {
		log.Fatalf("Error initializing security: %v", err)
	}

	// Setup server
	container := di.NewAppContainer(db, cfg)
	defer container.Close() // Ensure cleanup on shutdown

	runServer(setupRouter(container, cfg), cfg)
}

// runServer starts the HTTP server and blocks until an interrupt signal arrives.
func runServer(router http.Handler, cfg *config.Config) {
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
	router.Use(corsMiddleware(cfg))

	routes.RegisterPublicRoutes(router, container)
	routes.RegisterProtectedRoutes(router, container)
	routes.RegisterUtilityRoutes(router)

	log.Println("[Router]: Setup completed")

	return router
}

// setupDegradedRouter serves only the health check so the platform keeps the
// instance alive; every other route answers 503 with the reason.
func setupDegradedRouter(cfg *config.Config, reason string) *gin.Engine {
	middleware.UpdateHealthStatus("degraded")

	router := gin.Default()

	router.Use(middleware.RecoveryMiddleware())
	router.Use(corsMiddleware(cfg))

	routes.RegisterUtilityRoutes(router)

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Service is running in degraded mode",
			"details": reason,
		})
	})

	log.Printf("[Router]: Degraded setup completed (%s)", reason)

	return router
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowedOrigins,
		AllowMethods:     cfg.CORS.AllowedMethods,
		AllowHeaders:     cfg.CORS.AllowedHeaders,
		ExposeHeaders:    cfg.CORS.ExposedHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
	})
}

func generateEphemeralSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
