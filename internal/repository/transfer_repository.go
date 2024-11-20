package repository

import (
	"fmt"
	"log"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
)

func (r *Repository) HasItemsInLocation(itemIDs []int, fromLocationId int) (bool, error) {
	sql, args, err := r.GoguDBWrapper.Select(goqu.COUNT("id")).From("items").Where(goqu.Ex{
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

func (r *Repository) CanTransferNonSerializedItems(assets []models.UnserializedItemRequest, locationID int) (map[int]bool, error) {
	conditions := make([]goqu.Expression, 0, len(assets))
	for _, asset := range assets {
		conditions = append(conditions, goqu.And(
			goqu.C("item_category_id").Eq(asset.ItemCategoryID),
			goqu.C("location_id").Eq(locationID),
			goqu.C("quantity").Gte(asset.Quantity),
		))
	}

	sql, args, err := r.GoguDBWrapper.From("non_serialized_items").
		Select("item_category_id").
		Where(goqu.Or(conditions...)).
		ToSQL()

	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.DB.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	result := make(map[int]bool)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		result[id] = true
	}

	return result, nil
}

func (r *Repository) PerformTransfer(req models.TransferRequest, transitStatus string) (int, error) {
	var transferID int

	err := withTransaction(r.GoguDBWrapper, func(tx *goqu.TxDatabase) error {
		var err error
		if transferID, err = r.insertTransferRecord(tx, req); err != nil {
			return fmt.Errorf("failed to insert transfer record: %w", err)
		}

		if err = r.handleSerializedItems(tx, transferID, req.SerialziedItemCollection, req.LocationID, transitStatus); err != nil {
			return err
		}

		if err = r.handleNonSerializedItems(tx, transferID, req.UnserializedItemCollection, req.LocationID, req.FromLocationID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return transferID, nil
}

func (r *Repository) ConfirmTransfer(transferID string, status string) error {
	// TODO Transaction + remove transit status (do we really need this status?)
	query := r.GoguDBWrapper.
		Update("transfers").
		Set(goqu.Record{
			"status": status,
			// TODO "confirmed_at": goqu.L("NOW()"),
		}).
		Where(goqu.Ex{"id": transferID})

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to confirm transfer %s: %w", transferID, err)
	}

	return nil
}

func (r *Repository) RemoveAssetFromTransfer(transferID int, itemID int, locationID int) error {
	err := withTransaction(r.GoguDBWrapper, func(tx *goqu.TxDatabase) error {
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

func (r *Repository) moveSerializedItems(tx *goqu.TxDatabase, assets []int, locationID int, transitStatus string) error {
	locationCase := goqu.Case()
	transitStatusCase := goqu.Case()

	for _, asset := range assets {
		locationCase = locationCase.When(goqu.Ex{"id": asset}, locationID)
		transitStatusCase = transitStatusCase.When(goqu.Ex{"id": asset}, transitStatus)
	}

	query := tx.From("items").Update().
		Set(goqu.Record{
			"location_id": locationCase,
			"status":      transitStatusCase,
		}).
		Where(goqu.C("id").In(assets))

	if _, err := query.Executor().Exec(); err != nil {
		return fmt.Errorf("failed to update serialized assets: %w", err)
	}

	return nil
}

func (r *Repository) moveNonSerializedItems(tx *goqu.TxDatabase, unserializedItems []models.UnserializedItemRequest, toLocationID int, fromLocationID int) error {
	for _, unserializedItem := range unserializedItems {
		query := tx.Insert("non_serialized_items").
			Rows(goqu.Record{
				"item_category_id": unserializedItem.ItemCategoryID,
				"location_id":      toLocationID,
				"quantity":         unserializedItem.Quantity,
			}).
			OnConflict(
				goqu.DoUpdate(
					"item_category_id, location_id",
					goqu.Record{
						"quantity": goqu.L("non_serialized_items.quantity + EXCLUDED.quantity"),
					},
				),
			)

		if _, err := query.Executor().Exec(); err != nil {
			return fmt.Errorf("failed to upsert non-serialized asset for category %d: %w", unserializedItem.ItemCategoryID, err)
		}

		// Step 2: Decrease the quantity from the source location
		updateQuery := tx.Update("non_serialized_items").
			Set(goqu.Record{
				"quantity": goqu.L("quantity - ?", unserializedItem.Quantity),
			}).
			Where(goqu.Ex{
				"item_category_id": unserializedItem.ItemCategoryID,
				"location_id":      fromLocationID,
			}).
			Where(goqu.C("quantity").Gte(unserializedItem.Quantity)) // Ensure sufficient quantity

		result, err := updateQuery.Executor().Exec()
		if err != nil {
			return fmt.Errorf("failed to decrease quantity for category %d from location %d: %w", unserializedItem.ItemCategoryID, fromLocationID, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to check rows affected for category %d: %w", unserializedItem.ItemCategoryID, err)
		}

		if rowsAffected == 0 {
			return fmt.Errorf("insufficient quantity for category %d at location %d", unserializedItem.ItemCategoryID, fromLocationID)
		}
	}

	return nil
}

func (r *Repository) insertTransferRecord(tx *goqu.TxDatabase, req models.TransferRequest) (int, error) {
	query := tx.Insert("transfers").
		Rows(goqu.Record{
			"from_location_id": req.FromLocationID,
			"to_location_id":   req.LocationID,
			"status":           "in_transit",
		}).
		Returning("id")

	var transferID int
	if _, err := query.Executor().ScanVal(&transferID); err != nil {
		return 0, fmt.Errorf("failed to insert transfer record: %w", err)
	}

	return transferID, nil
}

func (r *Repository) insertSerializedItemTransferRecord(tx *goqu.TxDatabase, transferID int, assets []int) error {
	var records []goqu.Record
	for _, itemID := range assets {
		records = append(records, goqu.Record{
			"transfer_id": transferID,
			"item_id":     itemID,
		})
	}

	query := tx.Insert("serialized_transfers").Rows(records)

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to insert serialized asset transfers: %w", err)
	}

	return nil
}

func (r *Repository) insertNonSerializedItemTransferRecord(tx *goqu.TxDatabase, transferID int, unserializedItems []models.UnserializedItemRequest) error {
	var records []goqu.Record
	for _, unserializedItem := range unserializedItems {
		records = append(records, goqu.Record{
			"transfer_id":      transferID,
			"item_category_id": unserializedItem.ItemCategoryID,
			"quantity":         unserializedItem.Quantity,
		})
	}

	query := tx.Insert("non_serialized_transfers").Rows(records)

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to insert serialized asset transfers: %w", err)
	}

	return nil
}

func (r *Repository) handleSerializedItems(tx *goqu.TxDatabase, transferID int, assets []int, locationID int, transitStatus string) error {
	if len(assets) == 0 {
		return nil
	}

	if err := r.insertSerializedItemTransferRecord(tx, transferID, assets); err != nil {
		return fmt.Errorf("failed to insert serialized asset transfer record: %w", err)
	}

	if err := r.moveSerializedItems(tx, assets, locationID, transitStatus); err != nil {
		return fmt.Errorf("failed to move serialized assets: %w", err)
	}

	return nil
}

func (r *Repository) handleNonSerializedItems(tx *goqu.TxDatabase, transferID int, assets []models.UnserializedItemRequest, locationID, fromLocationID int) error {
	if len(assets) == 0 {
		return nil
	}

	if err := r.insertNonSerializedItemTransferRecord(tx, transferID, assets); err != nil {
		return fmt.Errorf("failed to insert non-serialized asset transfer record: %w", err)
	}

	if err := r.moveNonSerializedItems(tx, assets, locationID, fromLocationID); err != nil {
		return fmt.Errorf("failed to move non-serialized assets: %w", err)
	}

	return nil
}
