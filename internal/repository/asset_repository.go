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

func (r *Repository) HasItemsInLocation(itemIDs []int, fromLocationId int) (bool, error) {
	sql, args, err := r.goquDBWrapper.Select(goqu.COUNT("id")).From("items").Where(goqu.Ex{
		"location_id": fromLocationId,
		"id":          itemIDs,
	}).ToSQL()

	if err != nil {
		log.Fatalf("Failed to build query: %v", err)
	}

	var count int
	err = r.DB.QueryRow(sql, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to execute query: %w", err)
	}

	return count == len(itemIDs), nil
}

func (r *Repository) PersistItem(itemRequest models.ItemRequest) (*models.Asset, error) {
	query := r.goquDBWrapper.Insert("items").
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
	query := r.goquDBWrapper.
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

func (r *Repository) RemoveAssetFromTransfer(transferID int, itemID int, locationID int) error {
	err := withTransaction(r.goquDBWrapper, func(tx *goqu.TxDatabase) error {
		var err error
		_, err = tx.Delete("serialized_transfers").
			Where(goqu.Ex{
				"transfer_id": transferID,
				"item_id":     itemID,
			}).
			Executor().
			Exec()

		if err != nil {
			return fmt.Errorf("failed to remove asset from transfer %d: %w", transferID, err)
		}

		_, err = tx.Update("items").
			Set(goqu.Record{"location_id": locationID}).
			Where(goqu.Ex{"id": itemID}).
			Executor().
			Exec()

		if err != nil {
			return fmt.Errorf("failed to remove asset from transfer, unable to update location %d: %w", transferID, err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
