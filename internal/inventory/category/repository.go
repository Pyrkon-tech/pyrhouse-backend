package category

import (
	"fmt"
	"warehouse/internal/models"
	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
)

// CategoryRepository reprezentuje repozytorium dla kategorii
type CategoryRepository struct {
	repository *repository.Repository
}

// NewCategoryRepository tworzy nową instancję CategoryRepository
func NewCategoryRepository(repo *repository.Repository) *CategoryRepository {
	return &CategoryRepository{
		repository: repo,
	}
}

// CreateCategory tworzy nową kategorię
func (r *CategoryRepository) CreateCategory(category *models.ItemCategory) error {
	query := r.repository.GoquDBWrapper.Insert("item_category").
		Rows(goqu.Record{
			"label":  category.Label,
			"type":   category.Name,
			"pyr_id": category.PyrID,
		}).
		Returning("id")

	_, err := query.Executor().ScanVal(&category.ID)
	if err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}

	return nil
}

// GetCategoryByID pobiera kategorię po ID
func (r *CategoryRepository) GetCategoryByID(id int) (*models.ItemCategory, error) {
	var category models.ItemCategory
	query := r.repository.GoquDBWrapper.
		Select("*").
		From("item_category").
		Where(goqu.Ex{"id": id})

	_, err := query.Executor().ScanStruct(&category)
	if err != nil {
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	return &category, nil
}

// GetCategories pobiera listę kategorii
func (r *CategoryRepository) GetCategories() ([]models.ItemCategory, error) {
	var categories []models.ItemCategory
	query := r.repository.GoquDBWrapper.
		Select("*").
		From("item_category").
		Order(goqu.I("id").Asc())

	err := query.Executor().ScanStructs(&categories)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	return categories, nil
}

// UpdateCategory aktualizuje kategorię
func (r *CategoryRepository) UpdateCategory(category *models.ItemCategory) error {
	query := r.repository.GoquDBWrapper.
		Update("item_category").
		Set(goqu.Record{
			"label":  category.Label,
			"type":   category.Name,
			"pyr_id": category.PyrID,
		}).
		Where(goqu.Ex{"id": category.ID})

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	return nil
}

// DeleteCategory usuwa kategorię
func (r *CategoryRepository) DeleteCategory(id int) error {
	query := r.repository.GoquDBWrapper.
		Delete("item_category").
		Where(goqu.Ex{"id": id})

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	return nil
}
