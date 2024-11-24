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
	qb := goqu.Dialect("postgres")
	query := qb.Select("id", "item_category", "label").From("item_category")

	// rows, err := r.Query(query)
	sql, args, err := query.ToSQL()

	if err != nil {
		log.Fatalf("Failed to build query: %v", err)
	}
	rows, err := r.DB.Query(sql, args...)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}
	defer rows.Close()

	var itemCategories []models.ItemCategory
	for rows.Next() {
		var itemCategory models.ItemCategory
		if err := rows.Scan(&itemCategory.ID, &itemCategory.Type, &itemCategory.Label); err != nil {
			return nil, err
		}
		itemCategories = append(itemCategories, itemCategory)
	}
	return &itemCategories, err
}

func (r *Repository) PersistItemCategory(itemCategory models.ItemCategory) (*models.ItemCategory, error) {
	query := r.goquDBWrapper.Insert("item_category").
		Rows(goqu.Record{
			"item_category": itemCategory.Type,
			"label":         itemCategory.Label,
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
	result, err := r.goquDBWrapper.Delete("item_category").Where(goqu.Ex{"id": categoryID}).Executor().Exec()

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
