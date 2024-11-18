package repository

import (
	"fmt"
	"log"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
)

func (r *Repository) HasRelatedItems(categoryID string) bool {
	query := `SELECT COUNT(*) FROM items WHERE item_category_id = $1`
	var count int
	err := r.DB.QueryRow(query, categoryID).Scan(&count)
	if err != nil {
		log.Fatal("failed to check related items: ", err)

		return false
	}
	return count > 0
}

func (r *Repository) PersistItem(itemRequest models.ItemRequest) (*models.Item, error) {
	query := r.GoguDBWrapper.Insert("items").
		Rows(goqu.Record{
			"item_serial":      itemRequest.Serial,
			"location_id":      itemRequest.LocationId,
			"item_category_id": itemRequest.CategoryId,
		}).
		Returning("id")
	item := models.Item{
		Serial: itemRequest.Serial,
		Location: models.Location{
			ID: itemRequest.LocationId,
		},
		Category: models.ItemCategory{
			ID: itemRequest.CategoryId,
		},
	}

	if _, err := query.Executor().ScanVal(&item.ID); err != nil {
		return nil, fmt.Errorf("failed to insert item record: %w", err)
	}

	return &item, nil
}

func (r *Repository) UpdateItemStatus(itemIDs []int, status string) error {
	query := r.GoguDBWrapper.
		Update("items").
		Set(goqu.Record{
			"status": status,
		}).
		Where(goqu.Ex{"id": itemIDs})

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to confirm items transfer: %w", err)
	}

	return nil
}
