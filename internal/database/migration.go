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
	// Pobierz URL bazy danych z połączenia
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	// Utwórz logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Konwertuj ścieżkę na URL z prefiksem file://
	absPath, err := filepath.Abs(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	migrationsURL := "file://" + absPath

	// Uruchom migracje
	return migration.Migrate(dbURL, migrationsURL, true, logger)
}
