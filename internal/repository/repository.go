package repository

import (
	"database/sql"
	"fmt"

	"github.com/doug-martin/goqu/v9"
)

type Repository struct {
	DB            *sql.DB
	GoguDBWrapper *goqu.Database
}

// TODO remove db on migration period
func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		DB:            db,
		GoguDBWrapper: goqu.New("postgres", db),
	}
}

func withTransaction(db *goqu.Database, fn func(tx *goqu.TxDatabase) error) (err error) {
	// TODO Check refactor chat
	rawTx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	tx := goqu.NewTx("postgres", rawTx)
	defer func() {
		if p := recover(); p != nil {
			rawTx.Rollback()
			panic(p)
		} else if err != nil {
			rawTx.Rollback()
		} else {
			err = rawTx.Commit()
		}
	}()

	err = fn(tx)
	return
}
