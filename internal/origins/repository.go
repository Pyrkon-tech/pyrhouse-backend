package origins

import (
	"context"
	"fmt"
	"time"
	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
)

type Origin struct {
	ID          int       `json:"id" db:"id"`
	Slug        string    `json:"slug" db:"slug"`
	Label       string    `json:"label" db:"label"`
	AllowSuffix bool      `json:"allow_suffix" db:"allow_suffix"`
	Active      bool      `json:"active" db:"active"`
	SortOrder   int       `json:"sort_order" db:"sort_order"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type UpdateRequest struct {
	Label       *string `json:"label"`
	AllowSuffix *bool   `json:"allow_suffix"`
	Active      *bool   `json:"active"`
	SortOrder   *int    `json:"sort_order"`
}

type CreateRequest struct {
	Slug        string `json:"slug" binding:"required"`
	Label       string `json:"label" binding:"required"`
	AllowSuffix bool   `json:"allow_suffix"`
	SortOrder   int    `json:"sort_order"`
}

type Repository struct {
	repo *repository.Repository
}

func NewRepository(r *repository.Repository) *Repository {
	return &Repository{repo: r}
}

func (r *Repository) GetAll(_ context.Context) ([]Origin, error) {
	var origins []Origin
	query := r.repo.GoquDBWrapper.
		From("origins").
		Select("id", "slug", "label", "allow_suffix", "active", "sort_order", "created_at").
		Where(goqu.Ex{"active": true}).
		Order(goqu.I("sort_order").Asc())

	err := query.Executor().ScanStructs(&origins)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch origins: %w", err)
	}

	return origins, nil
}

func (r *Repository) GetAllIncludingInactive(_ context.Context) ([]Origin, error) {
	var origins []Origin
	query := r.repo.GoquDBWrapper.
		From("origins").
		Select("id", "slug", "label", "allow_suffix", "active", "sort_order", "created_at").
		Order(goqu.I("sort_order").Asc())

	err := query.Executor().ScanStructs(&origins)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch origins: %w", err)
	}

	return origins, nil
}

func (r *Repository) GetBySlug(_ context.Context, slug string) (*Origin, error) {
	var origin Origin
	query := r.repo.GoquDBWrapper.
		From("origins").
		Select("id", "slug", "label", "allow_suffix", "active", "sort_order", "created_at").
		Where(goqu.Ex{"slug": slug})

	found, err := query.Executor().ScanStruct(&origin)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch origin by slug: %w", err)
	}
	if !found {
		return nil, nil
	}

	return &origin, nil
}

func (r *Repository) GetByID(_ context.Context, id int) (*Origin, error) {
	var origin Origin
	query := r.repo.GoquDBWrapper.
		From("origins").
		Select("id", "slug", "label", "allow_suffix", "active", "sort_order", "created_at").
		Where(goqu.Ex{"id": id})

	found, err := query.Executor().ScanStruct(&origin)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch origin by id: %w", err)
	}
	if !found {
		return nil, nil
	}

	return &origin, nil
}

func (r *Repository) Create(_ context.Context, origin *Origin) error {
	record := goqu.Record{
		"slug":         origin.Slug,
		"label":        origin.Label,
		"allow_suffix": origin.AllowSuffix,
		"sort_order":   origin.SortOrder,
	}

	query := r.repo.GoquDBWrapper.
		Insert("origins").
		Rows(record).
		Returning("id", "active", "created_at")

	_, err := query.Executor().ScanStruct(origin)
	if err != nil {
		return fmt.Errorf("failed to create origin: %w", err)
	}

	return nil
}

func (r *Repository) Update(_ context.Context, id int, req UpdateRequest) (*Origin, error) {
	updates := goqu.Record{}
	if req.Label != nil {
		updates["label"] = *req.Label
	}
	if req.AllowSuffix != nil {
		updates["allow_suffix"] = *req.AllowSuffix
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	query := r.repo.GoquDBWrapper.
		Update("origins").
		Set(updates).
		Where(goqu.Ex{"id": id})

	result, err := query.Executor().Exec()
	if err != nil {
		return nil, fmt.Errorf("failed to update origin: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("origin not found")
	}

	return r.GetByID(context.Background(), id)
}

func (r *Repository) Deactivate(_ context.Context, id int) error {
	query := r.repo.GoquDBWrapper.
		Update("origins").
		Set(goqu.Record{"active": false}).
		Where(goqu.Ex{"id": id})

	result, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to deactivate origin: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("origin not found")
	}

	return nil
}
