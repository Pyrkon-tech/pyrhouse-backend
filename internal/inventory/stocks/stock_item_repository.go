package stocks

import (
	"fmt"
	"log"
	"warehouse/internal/repository"
	custom_error "warehouse/pkg/errors"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
	"github.com/lib/pq"
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
			"origin":           stockRequest.Origin,
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
		Origin: stockRequest.Origin,
	}

	if _, err := query.Executor().ScanVal(&stockItem.ID); err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			return nil, custom_error.WrapDBError("Duplicate serial number for asset", string(pqErr.Code))
		}
		return nil, fmt.Errorf("failed to insert stock item record: %w", err)
	}

	return &stockItem, nil
}

func (r *StockRepository) GetStockItems() (*[]models.StockItem, error) {
	var flatStocks []models.FlatStockRecord
	query := r.getStockItemQuery()

	err := query.Executor().ScanStructs(&flatStocks)

	if err != nil {
		return nil, fmt.Errorf("unable to select stock items from database: %s", err.Error())
	}
	var stocks []models.StockItem
	for _, flatStock := range flatStocks {
		stocks = append(stocks, transformToStockItem(flatStock))
	}

	return &stocks, nil
}

func (r *StockRepository) GetStockItemsBy(conditions repository.QueryBuilder) (*[]models.StockItem, error) {

	aliases := map[string]string{
		"location_ids":   "s.location_id",
		"category_id":    "s.item_category_id",
		"category_label": "c.label",
	}

	query := r.getStockItemQuery()
	query = query.
		Where(conditions.BuildConditions(aliases)).
		Order(goqu.I("s.id").Asc())

	var flatStocks []models.FlatStockRecord
	err := query.Executor().ScanStructs(&flatStocks)

	if err != nil {
		return nil, fmt.Errorf("unable to select stock items from database: %s", err.Error())
	}
	var stocks []models.StockItem
	for _, flatStock := range flatStocks {
		stocks = append(stocks, transformToStockItem(flatStock))
	}

	return &stocks, nil
}

func (r *StockRepository) GetStockItem(id int) (*models.StockItem, error) {
	var flatStock models.FlatStockRecord
	// Query to fetch flat stock data
	query := r.repository.GoquDBWrapper.
		Select(
			goqu.I("s.id").As("stock_id"),
			goqu.I("s.quantity").As("quantity"),
			goqu.I("s.origin").As("origin"),
			goqu.I("c.id").As("category_id"),
			goqu.I("c.item_category").As("category_type"),
			goqu.I("c.label").As("category_label"),
			goqu.I("c.pyr_id").As("category_pyr_id"),
			goqu.I("l.id").As("location_id"),
			goqu.I("l.name").As("location_name"),
		).
		From(goqu.T("non_serialized_items").As("s")).
		LeftJoin(
			goqu.T("item_category").As("c"),
			goqu.On(goqu.Ex{"s.item_category_id": goqu.I("c.id")}),
		).
		LeftJoin(
			goqu.T("locations").As("l"),
			goqu.On(goqu.Ex{"s.location_id": goqu.I("l.id")}),
		).
		Where(goqu.Ex{"s.id": id})

	_, err := query.Executor().ScanStruct(&flatStock)

	if err != nil {
		return nil, fmt.Errorf("unable to select stock items from database: %s", err.Error())
	}
	stock := transformToStockItem(flatStock)

	return &stock, nil
}

func (r *StockRepository) UpdateStock(stockRequest *PatchStockItemRequest) (*models.StockItem, error) {
	updates, err := buildUpdateFields(stockRequest)
	if err != nil {
		return nil, err
	}

	query := r.repository.GoquDBWrapper.
		Update("non_serialized_items").
		Set(updates).
		Where(goqu.Ex{"id": stockRequest.ID}) // Assuming `ID` is provided to identify the row

	result, err := query.Executor().Exec()
	if err != nil {
		return nil, fmt.Errorf("failed to update stock item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("no rows updated")
	}

	updatedStock, err := r.GetStockItem(stockRequest.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated stock item: %w", err)
	}

	return updatedStock, nil
}

func (r *StockRepository) GetStockItemsByTransfer(transferID int) (*[]models.StockItem, error) {
	var flatStocks []models.FlatStockRecord
	// Query to fetch flat stock data
	// TODO WRóć tu i zrób refactor zeby sie populowało
	query := r.repository.GoquDBWrapper.
		Select(
			goqu.I("nst.item_category_id").As("category_id"),
			goqu.I("nst.quantity").As("quantity"),
			goqu.I("nst.origin"),
			goqu.I("l.name").As("location_name"),
			goqu.I("c.id").As("category_id"),
			goqu.I("c.item_category").As("category_type"),
			goqu.I("c.label").As("category_label"),
			goqu.I("c.pyr_id").As("category_pyr_id"),
			goqu.I("nst.stock_id").As("transfer_stock_id"),
		).
		From(goqu.T("non_serialized_transfers").As("nst")).
		LeftJoin(
			goqu.T("transfers").As("t"),
			goqu.On(goqu.Ex{
				"t.id": transferID, // Directly use the value
			}),
		).
		LeftJoin(
			goqu.T("item_category").As("c"),
			goqu.On(goqu.Ex{"nst.item_category_id": goqu.I("c.id")}),
		).
		LeftJoin(
			goqu.T("locations").As("l"),
			goqu.On(goqu.Ex{"t.to_location_id": goqu.I("l.id")}),
		).
		Where(goqu.Ex{"nst.transfer_id": transferID})
	err := query.Executor().ScanStructs(&flatStocks)
	if err != nil {
		return nil, fmt.Errorf("error executing SQL statement for stock items: %w", err)
	}

	var stocks []models.StockItem
	for _, flatStock := range flatStocks {
		stocks = append(stocks, models.StockItem{
			ID: flatStock.TransferStockID,
			Category: models.ItemCategory{
				ID:    flatStock.CategoryID,
				Name:  flatStock.CategoryType,
				PyrID: flatStock.CategoryPyrId,
				Label: flatStock.CategoryLabel,
			},
			Location: models.Location{
				ID:   flatStock.LocationID,
				Name: flatStock.LocationName,
			},
			Quantity: flatStock.Quantity,
			Origin:   flatStock.Origin,
		})
	}

	return &stocks, nil
}

func (r *StockRepository) DecreaseStockItemsQuantity(tx *goqu.TxDatabase, stocks []models.StockItemRequest, fromLocationID int) error {
	for _, stockItem := range stocks {
		// Step 1: Decrease the quantity
		updateQuery := tx.Update("non_serialized_items").
			Set(goqu.Record{
				"quantity": goqu.L("quantity - ?", stockItem.Quantity),
			}).
			Where(goqu.Ex{
				"id":          stockItem.ID,
				"location_id": fromLocationID,
			}).
			Where(goqu.C("quantity").Gte(stockItem.Quantity)) // Ensure sufficient quantity

		result, err := updateQuery.Executor().Exec()
		if err != nil {
			return fmt.Errorf("failed to decrease quantity for category %d from location %d: %w", stockItem.ID, fromLocationID, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to check rows affected for category %d: %w", stockItem.ID, err)
		}

		if rowsAffected == 0 {
			return fmt.Errorf("insufficient quantity for category %d at location %d", stockItem.ID, fromLocationID)
		}

		if fromLocationID == 1 {
			log.Println("cannot decrease quantity form main warehouse")
			return nil
		}
		deleteQuery := tx.Delete("non_serialized_items").
			Where(goqu.Ex{
				"id":          stockItem.ID,
				"location_id": fromLocationID,
			}).
			Where(goqu.C("quantity").Eq(0)) // Only delete records where quantity is now zero

		if _, err := deleteQuery.Executor().Exec(); err != nil {
			return fmt.Errorf("failed to remove stock item with zero quantity: %w", err)
		}
	}

	return nil
}

func (r *StockRepository) IncreaseStockAtDestination(tx *goqu.TxDatabase, transferID int) error {
	query := `
		INSERT INTO non_serialized_items (item_category_id, location_id, quantity, origin)
		SELECT 
			nst.item_category_id, 
			t.to_location_id, 
			nst.quantity, 
			nst.origin
		FROM non_serialized_transfers nst
		INNER JOIN transfers t ON nst.transfer_id = t.id
		WHERE t.id = $1
		ON CONFLICT (item_category_id, location_id, origin)
		DO UPDATE SET quantity = non_serialized_items.quantity + EXCLUDED.quantity;
	`
	_, err := tx.Exec(query, transferID)
	if err != nil {
		return fmt.Errorf("failed to increase stock at destination: %w", err)
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

func (r *StockRepository) RestoreStockToLocation(tx *goqu.TxDatabase, transferReq RemoveStockItemFromTransferRequest, previousLocation int) error {
	_, err := tx.Update("non_serialized_items").
		Set(goqu.Record{"quantity": goqu.L("quantity + ?", transferReq.Quantity)}).
		Where(goqu.Ex{
			"item_category_id": transferReq.CategoryID,
			"location_id":      previousLocation,
		}).
		Executor().
		Exec()
	if err != nil {
		return fmt.Errorf("failed to restore stock to given location: %w", err)
	}

	return nil
}

func (r *StockRepository) getStockItemQuery() *goqu.SelectDataset {
	return r.repository.GoquDBWrapper.
		Select(
			goqu.I("s.id").As("stock_id"),
			goqu.I("s.quantity").As("quantity"),
			goqu.I("s.origin").As("origin"),
			goqu.I("c.id").As("category_id"),
			goqu.I("c.item_category").As("category_type"),
			goqu.I("c.label").As("category_label"),
			goqu.I("c.pyr_id").As("category_pyr_id"),
			goqu.I("c.category_type").As("category_equipment_type"),
			goqu.I("l.id").As("location_id"),
			goqu.I("l.name").As("location_name"),
		).
		From(goqu.T("non_serialized_items").As("s")).
		LeftJoin(
			goqu.T("item_category").As("c"),
			goqu.On(goqu.Ex{"s.item_category_id": goqu.I("c.id")}),
		).
		LeftJoin(
			goqu.T("locations").As("l"),
			goqu.On(goqu.Ex{"s.location_id": goqu.I("l.id")}),
		)
}

func transformToStockItem(flatStock models.FlatStockRecord) models.StockItem {
	return models.StockItem{
		ID:       flatStock.ID,
		Quantity: flatStock.Quantity,
		Origin:   flatStock.Origin,
		Category: models.ItemCategory{
			ID:    flatStock.CategoryID,
			Name:  flatStock.CategoryType,
			Label: flatStock.CategoryLabel,
			Type:  flatStock.CategoryEquipmentType,
		},
		Location: models.Location{
			ID:   flatStock.LocationID,
			Name: flatStock.LocationName,
		},
	}
}

func buildUpdateFields(stockRequest *PatchStockItemRequest) (goqu.Record, error) {
	updates := goqu.Record{}

	if stockRequest.Quantity != nil {
		updates["quantity"] = *stockRequest.Quantity
	}
	if stockRequest.Origin != nil {
		updates["origin"] = *stockRequest.Origin
	}
	if stockRequest.LocationID != nil {
		updates["location_id"] = *stockRequest.LocationID
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	return updates, nil
}
