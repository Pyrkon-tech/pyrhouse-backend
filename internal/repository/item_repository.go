package repository

import (
	"fmt"
	"log"
	custom_error "warehouse/pkg/errors"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
	"github.com/lib/pq"
)

func (r *Repository) HasRelatedItems(categoryID string) bool {
	query := `SELECT COUNT(*) FROM assets WHERE item_category_id = $1`
	var count int
	err := r.DB.QueryRow(query, categoryID).Scan(&count)
	if err != nil {
		log.Fatal("failed to check related assets: ", err)

		return false
	}
	return count > 0
}

func (r *Repository) PersistItem(itemRequest models.ItemRequest) (*models.Asset, error) {
	query := r.GoguDBWrapper.Insert("items").
		Rows(goqu.Record{
			"item_serial":      itemRequest.Serial,
			"location_id":      itemRequest.LocationId,
			"item_category_id": itemRequest.CategoryId,
		}).
		Returning("id")
	asset := models.Asset{
		Serial: itemRequest.Serial,
		Location: models.Location{
			ID: itemRequest.LocationId,
		},
		Category: models.ItemCategory{
			ID: itemRequest.CategoryId,
		},
	}

	if _, err := query.Executor().ScanVal(&asset.ID); err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return nil, custom_error.WrapDBError("Duplicate serial number for asset", string(pqErr.Code))
			}
		}
		return nil, fmt.Errorf("failed to insert asset record: %w", err)
	}

	return &asset, nil
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
		return fmt.Errorf("failed to confirm assets transfer: %w", err)
	}

	return nil
}
