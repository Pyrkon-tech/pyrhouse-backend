package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"warehouse/internal/config"

	_ "github.com/lib/pq"
)

// NewPostgresConnection opens the pool and verifies it with a bounded ping.
func NewPostgresConnection(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("could not connect to postgres: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("could not ping the database: %w", err)
	}

	return db, nil
}

// ConnectWithRetry keeps retrying NewPostgresConnection with an exponential
// backoff. A managed database is not always reachable the moment a new instance
// boots (failover, connection pool restart, deploy race), and a crash there
// costs a full container restart cycle.
func ConnectWithRetry(cfg config.DatabaseConfig) (*sql.DB, error) {
	attempts := cfg.ConnectRetries
	if attempts < 1 {
		attempts = 1
	}

	backoff := time.Second
	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		db, err := NewPostgresConnection(cfg)
		if err == nil {
			if attempt > 1 {
				log.Printf("[DB]: Connected after %d attempts", attempt)
			}
			return db, nil
		}

		lastErr = err
		if attempt == attempts {
			break
		}

		log.Printf("[DB]: Connection attempt %d/%d failed: %v - retrying in %v", attempt, attempts, err, backoff)
		time.Sleep(backoff)
		if backoff < 8*time.Second {
			backoff *= 2
		}
	}

	return nil, fmt.Errorf("database unreachable after %d attempts: %w", attempts, lastErr)
}
