package main

import (
	"context"
	"flag"
	"fmt"
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

	// Error reporting first, so failures below are reported and not just logged.
	// A missing SENTRY_DSN simply leaves reporting off.
	if err := middleware.InitSentry(middleware.SentryOptions{
		DSN:              cfg.Sentry.DSN,
		Environment:      cfg.Sentry.Environment,
		Release:          cfg.Server.Version,
		TracesSampleRate: cfg.Sentry.TracesSampleRate,
	}); err != nil {
		log.Printf("[WARN] Sentry initialization failed: %v", err)
	}
	defer middleware.FlushSentry()

	// Validate required config. JWT_SECRET is checked later - migrations do not
	// need it, and `main -migrate` runs before the app in start.sh and in CI.
	if cfg.Database.URL == "" {
		fatalf("DATABASE_URL environment variable is not set")
	}

	// Parse command line flags
	migrateOnly := flag.Bool("migrate", false, "run only migrations without starting the server")
	migrationsDir := flag.String("dir", "./migrations", "directory containing migration files")
	flag.Parse()

	// Setup DB - retried, because a managed database is not always reachable the
	// moment a new instance boots.
	db, err := database.ConnectWithRetry(cfg.Database)
	if err != nil {
		fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()
	log.Println("[DB]: Setup completed")

	// Run migrations if requested
	if *migrateOnly {
		if err := database.RunMigrations(db, *migrationsDir); err != nil {
			fatalf("Error running migrations: %v", err)
		}
		log.Println("[Migrations]: Completed successfully")
		return
	}

	// Initialize security module
	if cfg.JWT.Secret == "" {
		fatalf("JWT_SECRET environment variable is not set")
	}
	if err := security.Initialize(cfg.JWT); err != nil {
		fatalf("Error initializing security: %v", err)
	}

	// Setup server
	container := di.NewAppContainer(db, cfg)
	defer container.Close() // Ensure cleanup on shutdown

	middleware.SetVersion(cfg.Server.Version)
	middleware.SetDatabaseChecker(db.PingContext)

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
			fatalf("Failed to start server: %v", err)
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
		fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

func setupRouter(container *di.Container, cfg *config.Config) *gin.Engine {
	router := gin.Default()

	router.Use(middleware.RecoveryMiddleware())

	// Registered after RecoveryMiddleware so a panic reported to Sentry is still
	// turned into the standard JSON 500 by the recovery above.
	for _, handler := range middleware.SentryHandlers() {
		router.Use(handler)
	}

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

// fatalf reports the failure to Sentry, flushes, and exits. A boot that never
// finishes is the failure most worth seeing, and log.Fatal skips deferred work.
func fatalf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	middleware.CaptureFatal(msg)
	log.Fatal(msg)
}
