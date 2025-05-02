package repository

import (
	"fmt"
	custom_error "warehouse/pkg/errors"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
	"github.com/lib/pq"
)

func (r *Repository) GetCategories() (*[]models.ItemCategory, error) {
	var categories []models.ItemCategory
	query := r.GoquDBWrapper.Select(
		goqu.I("id").As("category_id"),
		goqu.I("item_category").As("type"),
		goqu.I("category_type").As("category_type"),
		goqu.I("label"),
		goqu.I("pyr_id"),
	).
		From("item_category")

	err := query.Executor().ScanStructs(&categories)

	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}

	return &categories, err
}

func (r *Repository) PersistItemCategory(itemCategory models.ItemCategory) (*models.ItemCategory, error) {
	query := r.GoquDBWrapper.Insert("item_category").
		Rows(goqu.Record{
			"item_category": itemCategory.Name,
			"label":         itemCategory.Label,
			"pyr_id":        itemCategory.PyrID,
			"category_type": itemCategory.Type,
		}).
		Returning("id")

	if _, err := query.Executor().ScanVal(&itemCategory.ID); err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return nil, custom_error.WrapDBError("Duplicate serial number for asset", string(pqErr.Code))
			}
		}
		return nil, fmt.Errorf("failed to insert item_category record: %w", err)
	}

	return &itemCategory, nil
}

func (r *Repository) DeleteItemCategoryByID(categoryID string) error {
	// Sprawdź czy są przypisane elementy w non_serialized_items
	var nonSerializedCount int
	query := r.GoquDBWrapper.Select(goqu.COUNT("*")).
		From("non_serialized_items").
		Where(goqu.Ex{"item_category_id": categoryID})

	_, err := query.Executor().ScanVal(&nonSerializedCount)
	if err != nil {
		return fmt.Errorf("failed to check if category has non-serialized items: %w", err)
	}

	// Sprawdź czy są przypisane elementy w items
	var itemsCount int
	query = r.GoquDBWrapper.Select(goqu.COUNT("*")).
		From("items").
		Where(goqu.Ex{"item_category_id": categoryID})

	_, err = query.Executor().ScanVal(&itemsCount)
	if err != nil {
		return fmt.Errorf("failed to check if category has items: %w", err)
	}

	if nonSerializedCount > 0 || itemsCount > 0 {
		return custom_error.WrapDBError(
			"Nie można usunąć kategorii, ponieważ ma przypisane elementy",
			"23503",
		)
	}

	result, err := r.GoquDBWrapper.Delete("item_category").
		Where(goqu.Ex{"id": categoryID}).
		Executor().
		Exec()

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23503" { // foreign key violation
				return custom_error.WrapDBError(
					"Nie można usunąć kategorii, ponieważ ma przypisane elementy",
					string(pqErr.Code),
				)
			}
		}
		return fmt.Errorf("failed to delete category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not retrieve rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no category found with id: %s", categoryID)
	}

	return nil
}

func (r *Repository) GetCategoryType(ID int) (string, error) {
	var categoryType string
	query := r.GoquDBWrapper.Select(goqu.I("category_type").As("category_type")).
		From("item_category").
		Where(goqu.Ex{"id": ID})

	_, err := query.Executor().ScanVal(&categoryType)

	if err != nil {
		return "", fmt.Errorf("failed to query categories: %w", err)
	}

	return categoryType, err
}

func (r *Repository) UpdateItemCategory(categoryID int, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	query := r.GoquDBWrapper.Update("item_category").
		Set(updates).
		Where(goqu.Ex{"id": categoryID})

	result, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to update item_category record: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not retrieve rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no category found with id: %d", categoryID)
	}

	return nil
}
