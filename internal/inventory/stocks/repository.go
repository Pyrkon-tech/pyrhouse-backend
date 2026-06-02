package stocks

import (
	"fmt"
	apperrors "warehouse/internal/errors"
	"warehouse/internal/models"
	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
	"github.com/lib/pq"
)

type StockRepository struct {
	repository *repository.Repository
}

func NewRepository(r *repository.Repository) *StockRepository {
	return &StockRepository{repository: r}
}

func (r *StockRepository) PersistStockItem(stockRequest models.CreateStockItemRequest) (*models.StockItem, error) {
	sql := `
		INSERT INTO non_serialized_items (item_category_id, location_id, quantity, origin_id, origin_suffix)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (item_category_id, location_id, origin_id, origin_suffix)
		DO UPDATE SET quantity = non_serialized_items.quantity + EXCLUDED.quantity
		RETURNING id, quantity
	`
	var id, quantity int
	err := r.repository.DB.QueryRow(sql, stockRequest.CategoryID, stockRequest.LocationID, stockRequest.Quantity, stockRequest.OriginID, stockRequest.OriginSuffix).Scan(&id, &quantity)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if string(pqErr.Code) == "23503" {
				return nil, fmt.Errorf("invalid location_id or category_id: referenced record does not exist")
			}
			return nil, apperrors.WrapDBError(pqErr.Message, string(pqErr.Code))
		}
		return nil, fmt.Errorf("failed to insert stock item record: %w", err)
	}

	return &models.StockItem{
		ID:       id,
		Quantity: quantity,
		Category: models.ItemCategory{ID: stockRequest.CategoryID},
		Location: models.Location{ID: stockRequest.LocationID},
		Origin:   stockRequest.Origin,
	}, nil
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
		"location_id":    "s.location_id",
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
	query := r.getStockItemQuery().Where(goqu.Ex{"s.id": id})

	_, err := query.Executor().ScanStruct(&flatStock)

	if err != nil {
		return nil, fmt.Errorf("unable to select stock items from database: %s", err.Error())
	}
	stock := transformToStockItem(flatStock)

	return &stock, nil
}

func (r *StockRepository) HasRelatedItems(categoryID string) (bool, error) {
	query := `SELECT COUNT(*) FROM non_serialized_items WHERE item_category_id = $1`
	var count int
	err := r.repository.DB.QueryRow(query, categoryID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check related stock items: %w", err)
	}
	return count > 0, nil
}

func (r *StockRepository) UpdateStock(stockRequest *models.PatchStockItemRequest) (*models.StockItem, error) {
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

	query := r.repository.GoquDBWrapper.
		Select(
			goqu.I("nst.item_category_id").As("category_id"),
			goqu.I("nst.quantity").As("quantity"),
			goqu.L("COALESCE(CASE WHEN nst.origin_suffix IS NOT NULL THEN o.slug || '-' || nst.origin_suffix ELSE o.slug END, '')").As("origin"),
			goqu.I("l.name").As("location_name"),
			goqu.I("c.id").As("category_id"),
			goqu.I("c.item_category").As("category_type"),
			goqu.I("c.label").As("category_label"),
			goqu.I("c.pyr_id").As("category_pyr_id"),
			goqu.I("nst.stock_id").As("transfer_stock_id"),
		).
		From(goqu.T("non_serialized_transfers").As("nst")).
		InnerJoin(
			goqu.T("transfers").As("t"),
			goqu.On(goqu.Ex{"t.id": transferID}),
		).
		LeftJoin(
			goqu.T("origins").As("o"),
			goqu.On(goqu.Ex{"nst.origin_id": goqu.I("o.id")}),
		).
		InnerJoin(
			goqu.T("item_category").As("c"),
			goqu.On(goqu.Ex{"nst.item_category_id": goqu.I("c.id")}),
		).
		InnerJoin(
			goqu.T("locations").As("l"),
			goqu.On(goqu.Ex{"t.to_location_id": goqu.I("l.id")}),
		).
		Where(goqu.Ex{"nst.transfer_id": transferID})

	err := query.Executor().ScanStructs(&flatStocks)
	if err != nil {
		return nil, fmt.Errorf("error executing SQL statement for stock items: %w", err)
	}

	stocks := make([]models.StockItem, len(flatStocks))
	for i, flatStock := range flatStocks {
		stocks[i] = models.StockItem{
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
		}
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
			continue
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
		INSERT INTO non_serialized_items (item_category_id, location_id, quantity, origin_id, origin_suffix)
		SELECT
			nst.item_category_id,
			t.to_location_id,
			nst.quantity,
			nst.origin_id,
			nst.origin_suffix
		FROM non_serialized_transfers nst
		INNER JOIN transfers t ON nst.transfer_id = t.id
		WHERE t.id = $1
		ON CONFLICT (item_category_id, location_id, origin_id, origin_suffix)
		DO UPDATE SET quantity = non_serialized_items.quantity + EXCLUDED.quantity;
	`
	_, err := tx.Exec(query, transferID)
	if err != nil {
		return fmt.Errorf("failed to increase stock at destination: %w", err)
	}

	return nil
}

func (r *StockRepository) RemoveZeroQuantityStock(tx *goqu.TxDatabase, transferReq models.RemoveStockItemFromTransferRequest) error {
	query := tx.Delete("non_serialized_items").
		Where(goqu.Ex{
			"item_category_id": transferReq.CategoryID,
			"location_id":      transferReq.ToLocationID,
		})

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to remove zero quantity stock: %w", err)
	}

	return nil
}

// RestoreStockToLocation returns quantity for a specific category back to the transfer's source location.
// Used when removing a single item from an in-transit transfer (partial removal).
func (r *StockRepository) RestoreStockToLocation(tx *goqu.TxDatabase, transferReq models.RemoveStockItemFromTransferRequest) error {
	query := `
		INSERT INTO non_serialized_items (item_category_id, location_id, quantity, origin_id, origin_suffix)
		SELECT nst.item_category_id, t.from_location_id, $1, nst.origin_id, nst.origin_suffix
		FROM non_serialized_transfers nst
		INNER JOIN transfers t ON t.id = nst.transfer_id
		WHERE nst.transfer_id = $2 AND nst.item_category_id = $3
		LIMIT 1
		ON CONFLICT (item_category_id, location_id, origin_id, origin_suffix)
		DO UPDATE SET quantity = non_serialized_items.quantity + EXCLUDED.quantity
	`
	_, err := tx.Exec(query, transferReq.Quantity, transferReq.TransferID, transferReq.CategoryID)
	if err != nil {
		return fmt.Errorf("failed to restore stock to location: %w", err)
	}

	return nil
}

// RestoreStockFromCancelledTransfer returns all transferred quantities back to from_location.
// Symmetric to IncreaseStockAtDestination — used on transfer cancellation.
func (r *StockRepository) RestoreStockFromCancelledTransfer(tx *goqu.TxDatabase, transferID int) error {
	query := `
		INSERT INTO non_serialized_items (item_category_id, location_id, quantity, origin_id, origin_suffix)
		SELECT nst.item_category_id, t.from_location_id, nst.quantity, nst.origin_id, nst.origin_suffix
		FROM non_serialized_transfers nst
		INNER JOIN transfers t ON nst.transfer_id = t.id
		WHERE nst.transfer_id = $1
		ON CONFLICT (item_category_id, location_id, origin_id, origin_suffix)
		DO UPDATE SET quantity = non_serialized_items.quantity + EXCLUDED.quantity
	`
	_, err := tx.Exec(query, transferID)
	if err != nil {
		return fmt.Errorf("failed to restore stock from cancelled transfer: %w", err)
	}

	return nil
}

func (r *StockRepository) DeleteStock(id int) error {
	_, err := r.repository.GoquDBWrapper.Delete("non_serialized_items").
		Where(goqu.Ex{"id": id}).
		Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to delete stock: %w", err)
	}

	return nil
}

func (r *StockRepository) getStockItemQuery() *goqu.SelectDataset {
	return r.repository.GoquDBWrapper.
		Select(
			goqu.I("s.id").As("stock_id"),
			goqu.I("s.quantity").As("quantity"),
			goqu.L("CASE WHEN s.origin_suffix IS NOT NULL THEN o.slug || '-' || s.origin_suffix ELSE o.slug END").As("origin"),
			goqu.I("c.id").As("category_id"),
			goqu.I("c.item_category").As("category_type"),
			goqu.I("c.label").As("category_label"),
			goqu.I("c.pyr_id").As("category_pyr_id"),
			goqu.I("c.category_type").As("category_equipment_type"),
			goqu.I("l.id").As("location_id"),
			goqu.I("l.name").As("location_name"),
			goqu.I("l.pavilion").As("location_pavilion"),
		).
		From(goqu.T("non_serialized_items").As("s")).
		LeftJoin(
			goqu.T("origins").As("o"),
			goqu.On(goqu.Ex{"s.origin_id": goqu.I("o.id")}),
		).
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
			ID:       flatStock.LocationID,
			Name:     flatStock.LocationName,
			Pavilion: flatStock.LocationPavilion,
		},
	}
}

func buildUpdateFields(stockRequest *models.PatchStockItemRequest) (goqu.Record, error) {
	updates := goqu.Record{}

	if stockRequest.Quantity != nil {
		updates["quantity"] = *stockRequest.Quantity
	}
	if stockRequest.OriginID != nil {
		updates["origin_id"] = *stockRequest.OriginID
		updates["origin_suffix"] = stockRequest.OriginSuffix
	}
	if stockRequest.LocationID != nil {
		updates["location_id"] = *stockRequest.LocationID
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	return updates, nil
}
