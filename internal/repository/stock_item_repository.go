package repository

import (
	"fmt"
	stock_request "warehouse/internal/stocks/request"
	transfer_request "warehouse/internal/transfers/request"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
)

func (r *Repository) PersistStockItem(stockRequest stock_request.StockItemRequest) (*models.StockItem, error) {
	query := r.goquDBWrapper.Insert("non_serialized_items").
		Rows(goqu.Record{
			"quantity":         stockRequest.Quantity,
			"location_id":      stockRequest.LocationID,
			"item_category_id": stockRequest.CategoryID,
		}).
		Returning("id")
	stockItem := models.StockItem{
		Quantity: stockRequest.Quantity,
		Category: models.ItemCategory{
			ID: stockRequest.CategoryID,
		},
		Location: models.Location{
			ID: stockRequest.LocationID,
		},
	}

	if _, err := query.Executor().ScanVal(&stockItem.ID); err != nil {
		return nil, fmt.Errorf("failed to insert stock item record: %w", err)
	}

	return &stockItem, nil
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

func (r *Repository) RemoveStockItemFromTransfer(transferReq transfer_request.RemoveStockItemFromTransferRequest) error {
	return withTransaction(r.goquDBWrapper, func(tx *goqu.TxDatabase) error {
		if err := decreaseStockInTransfer(tx, transferReq); err != nil {
			return err
		}

		if err := removeZeroQuantityStock(r.goquDBWrapper, transferReq); err != nil {
			return err
		}

		var previousLocation int
		_, err := tx.Select("to_location_id").From("transfers").Where(goqu.Ex{"id": transferReq.TransferID}).Executor().ScanVal(&previousLocation)
		if err != nil {
			return fmt.Errorf("failed to fetch to_location_id: %w", err)
		}

		if err := restoreStockToLocation(tx, transferReq, previousLocation); err != nil {
			return err
		}

		return nil
	})
}

func decreaseStockInTransfer(tx *goqu.TxDatabase, transferReq transfer_request.RemoveStockItemFromTransferRequest) error {
	updateResult, err := tx.Update("non_serialized_transfers").
		Set(goqu.Record{"quantity": goqu.L("quantity - ?", transferReq.Quantity)}).
		Where(goqu.Ex{
			"transfer_id":      transferReq.TransferID,
			"item_category_id": transferReq.CategoryID,
		}).
		Where(goqu.C("quantity").Gte(transferReq.Quantity)).
		Executor().
		Exec()
	if err != nil {
		return fmt.Errorf("failed to lower stock from transfer %d: %w", transferReq.TransferID, err)
	}

	rowsAffected, err := updateResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("insufficient stock for item_category_id %d at location %d", transferReq.CategoryID, transferReq.LocationID)
	}

	return nil
}

func removeZeroQuantityStock(dbWrapper *goqu.Database, transferReq transfer_request.RemoveStockItemFromTransferRequest) error {
	deleteQuery := dbWrapper.Delete("non_serialized_transfers").
		Where(goqu.Ex{
			"item_category_id": transferReq.CategoryID,
			"transfer_id":      transferReq.TransferID,
		}).
		Where(goqu.C("quantity").Eq(0))

	if _, err := deleteQuery.Executor().Exec(); err != nil {
		return fmt.Errorf("failed to remove stock item with zero quantity: %w", err)
	}

	return nil
}

// TODO add handling to remove from previous location
func restoreStockToLocation(tx *goqu.TxDatabase, transferReq transfer_request.RemoveStockItemFromTransferRequest, previousLocation int) error {
	_, err := tx.Update("non_serialized_items").
		Set(goqu.Record{"quantity": goqu.L("quantity + ?", transferReq.Quantity)}).
		Where(goqu.Ex{
			"item_category_id": transferReq.CategoryID,
			"location_id":      transferReq.LocationID,
		}).
		Executor().
		Exec()
	if err != nil {
		return fmt.Errorf("failed to restore stock to given location: %w", err)
	}

	_, err = tx.Update("non_serialized_items").
		Set(goqu.Record{"quantity": goqu.L("quantity - ?", transferReq.Quantity)}).
		Where(goqu.Ex{
			"item_category_id": transferReq.CategoryID,
			"location_id":      previousLocation,
		}).
		Executor().
		Exec()
	if err != nil {
		return fmt.Errorf("failed to restore stock from previous location: %w", err)
	}

	return nil
}
