package repository

import (
	"database/sql"
	"fmt"

	"github.com/doug-martin/goqu/v9"
)

// TODO lower case, no need for public
type Repository struct {
	DB            *sql.DB
	GoquDBWrapper *goqu.Database
}

// TODO remove db on migration period
func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		DB:            db,
		GoquDBWrapper: goqu.New("postgres", db),
	}
}

func WithTransaction(db *goqu.Database, fn func(*goqu.TxDatabase) error) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rollback err: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
