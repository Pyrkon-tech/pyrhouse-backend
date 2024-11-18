package repository

import (
	"database/sql"

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
