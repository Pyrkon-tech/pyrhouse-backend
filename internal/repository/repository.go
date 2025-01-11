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

func WithTransaction(db *goqu.Database, fn func(tx *goqu.TxDatabase) error) (err error) {
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
