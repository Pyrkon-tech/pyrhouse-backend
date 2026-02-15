package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"warehouse/internal/database/migration"

	"go.uber.org/zap"
)

func RunMigrations(db *sql.DB, migrationsDir string) error {
	// Get the database URL from the connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	// Create logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Convert the path to a URL with file:// prefix
	absPath, err := filepath.Abs(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	migrationsURL := "file://" + absPath

	// Run migrations
	return migration.Migrate(dbURL, migrationsURL, true, logger)
}
