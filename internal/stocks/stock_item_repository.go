package stocks

import (
	"fmt"
	"warehouse/internal/repository"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
)

type StockRepository struct {
	repository *repository.Repository
}

func NewRepository(r *repository.Repository) *StockRepository {
	return &StockRepository{repository: r}
}

func (r *StockRepository) PersistStockItem(stockRequest StockItemRequest) (*models.StockItem, error) {
	query := r.repository.GoquDBWrapper.Insert("non_serialized_items").
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

func (r *StockRepository) GetStockItemsByTransfer(transferID int) (*[]models.StockItem, error) {
	var flatStocks []models.StockItemFlat
	// Query to fetch flat stock data
	query := r.repository.GoquDBWrapper.
		Select(
			goqu.I("s.id").As("stock_id"),
			goqu.I("nst.item_category_id").As("category_id"),
			goqu.I("nst.quantity").As("quantity"),
		).
		From(goqu.T("non_serialized_transfers").As("nst")).
		LeftJoin(
			goqu.T("transfers").As("t"),
			goqu.On(goqu.Ex{
				"t.id": transferID, // Directly use the value
			}),
		).
		LeftJoin(
			goqu.T("non_serialized_items").As("s"),
			goqu.On(goqu.Ex{
				"s.location_id":      goqu.I("t.to_location_id"),
				"s.item_category_id": goqu.I("nst.item_category_id"),
			}),
		).
		Where(goqu.Ex{"nst.transfer_id": transferID})
	err := query.Executor().ScanStructs(&flatStocks)
	if err != nil {
		return nil, fmt.Errorf("error executing SQL statement for stock items: %w", err)
	}

	var stocks []models.StockItem
	for _, flatStock := range flatStocks {
		stocks = append(stocks, models.StockItem{
			ID: flatStock.ID,
			Category: models.ItemCategory{
				ID: flatStock.CategoryID,
			},
			Quantity: flatStock.Quantity,
		})
	}

	return &stocks, nil
}

func (r *StockRepository) MoveNonSerializedItems(tx *goqu.TxDatabase, unserializedItems []models.UnserializedItemRequest, toLocationID int, fromLocationID int) error {
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

func (r *StockRepository) RemoveZeroQuantityStock(tx *goqu.TxDatabase, transferReq RemoveStockItemFromTransferRequest) error {
	deleteQuery := tx.Delete("non_serialized_transfers").
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
func RestoreStockToLocation(tx *goqu.TxDatabase, transferReq RemoveStockItemFromTransferRequest, previousLocation int) error {
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
