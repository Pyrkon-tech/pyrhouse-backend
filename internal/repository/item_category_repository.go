package repository

import (
	"fmt"
	"log"
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
	result, err := r.GoquDBWrapper.Delete("item_category").Where(goqu.Ex{"id": categoryID}).Executor().Exec()

	if err != nil {
		log.Fatal("failed to delete asset category: ", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not retrieve rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no asset category found with id: %s", categoryID)
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
