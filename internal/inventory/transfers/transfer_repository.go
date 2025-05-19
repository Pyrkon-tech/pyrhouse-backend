package transfers

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	"warehouse/internal/inventory/stocks"
	"warehouse/internal/repository"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
)

type TransferRepository interface {
	CanTransferNonSerializedItems(assets []models.StockItemRequest, locationID int) (map[int]bool, error)
	UpdateTransferStatus(transferID int, status string) error
	GetTransferRow(transferID int) (*FlatTransfer, error)
	GetTransferRows(conditions repository.QueryBuilder) (*[]FlatTransfer, error)
	GetTransfersByUserAndStatus(userID int, status string) ([]FlatTransfer, error)
	InsertTransferRecord(tx *goqu.TxDatabase, req models.TransferRequest) (int, error)
	GetTransferLocationById(tx *goqu.TxDatabase, transferID int) (int, error)
	InsertAssetsTransferRecord(tx *goqu.TxDatabase, transferID int, assets []int) error
	MoveAssets(tx *goqu.TxDatabase, assets []int, locationID int, transitStatus string) error
	InsertStockItemsTransferRecord(tx *goqu.TxDatabase, transferID int, unserializedItems []models.StockItemRequest) error
	RemoveStockItemsTransferRecords(tx *goqu.TxDatabase, transferID int) error
	HasStockItemsInTransfer(tx *goqu.TxDatabase, transferID int) (bool, error)
	InsertTransferUsers(tx *goqu.TxDatabase, transferID int, users []models.TransferUser) error
	GetTransferUsers(transferID int) ([]models.User, error)
	UpdateDeliveryLocation(transferID int, latitude float64, longitude float64, timestamp time.Time) error
	UpdateStockItemsTransferStatus(tx *goqu.TxDatabase, transferID int, status string) error
	SetTransferUsers(transferID int, userIDs []int) error
}

type transferRepository struct {
	Repo *repository.Repository
}

func NewRepository(r *repository.Repository) *transferRepository {
	return &transferRepository{Repo: r}
}

func (r *transferRepository) CanTransferNonSerializedItems(stocks []models.StockItemRequest, locationID int) (map[int]bool, error) {
	conditions := make([]goqu.Expression, 0, len(stocks))
	for _, stockItem := range stocks {
		conditions = append(conditions, goqu.And(
			goqu.C("id").Eq(stockItem.ID),
			goqu.C("location_id").Eq(locationID),
			goqu.C("quantity").Gte(stockItem.Quantity),
		))
	}

	sql, args, err := r.Repo.GoquDBWrapper.From("non_serialized_items").
		Select("id").
		Where(goqu.Or(conditions...)).
		ToSQL()

	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.Repo.DB.Query(sql, args...)
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

type FlatTransfer struct {
	ID                   int            `db:"transfer_id"`
	FromLocationID       int            `db:"from_location_id"`
	FromLocationName     string         `db:"from_location_name"`
	FromLocationPavilion sql.NullString `db:"from_location_pavilion"`
	ToLocationID         int            `db:"to_location_id"`
	ToLocationName       string         `db:"to_location_name"`
	ToLocationPavilion   sql.NullString `db:"to_location_pavilion"`
	TransferDate         time.Time      `db:"transfer_date"`
	Status               string         `db:"transfer_status"`
	DeliveryLatitude     *float64       `db:"delivery_latitude"`
	DeliveryLongitude    *float64       `db:"delivery_longitude"`
	DeliveryTimestamp    *time.Time     `db:"delivery_timestamp"`
}

func (r *transferRepository) GetTransferRow(transferID int) (*FlatTransfer, error) {
	var transfer FlatTransfer

	query := r.Repo.GoquDBWrapper.
		Select(
			goqu.I("t.id").As("transfer_id"),
			goqu.I("l1.id").As("from_location_id"),
			goqu.I("l1.name").As("from_location_name"),
			goqu.I("l1.pavilion").As("from_location_pavilion"),
			goqu.I("l2.id").As("to_location_id"),
			goqu.I("l2.name").As("to_location_name"),
			goqu.I("l2.pavilion").As("to_location_pavilion"),
			goqu.I("t.status").As("transfer_status"),
			goqu.I("t.transfer_date").As("transfer_date"),
			goqu.I("t.delivery_latitude").As("delivery_latitude"),
			goqu.I("t.delivery_longitude").As("delivery_longitude"),
			goqu.I("t.delivery_timestamp").As("delivery_timestamp"),
		).
		From(goqu.T("transfers").As("t")).
		LeftJoin(
			goqu.T("locations").As("l1"),
			goqu.On(goqu.Ex{"t.from_location_id": goqu.I("l1.id")}),
		).
		LeftJoin(
			goqu.T("locations").As("l2"),
			goqu.On(goqu.Ex{"t.to_location_id": goqu.I("l2.id")}),
		).
		Where(goqu.Ex{"t.id": transferID})

	_, err := query.Executor().ScanStruct(&transfer)
	if err != nil {
		return nil, fmt.Errorf("error executing SQL statement: %w", err)
	}

	return &transfer, nil
}

func (r *transferRepository) GetTransferRows(conditions repository.QueryBuilder) (*[]FlatTransfer, error) {
	var flatTransfers []FlatTransfer

	query := r.Repo.GoquDBWrapper.
		Select(
			goqu.I("t.id").As("transfer_id"),
			goqu.I("l1.id").As("from_location_id"),
			goqu.I("l1.name").As("from_location_name"),
			goqu.I("l1.pavilion").As("from_location_pavilion"),
			goqu.I("l2.id").As("to_location_id"),
			goqu.I("l2.name").As("to_location_name"),
			goqu.I("l2.pavilion").As("to_location_pavilion"),
			goqu.I("t.status").As("transfer_status"),
			goqu.I("t.transfer_date").As("transfer_date"),
		).
		From(goqu.T("transfers").As("t")).
		LeftJoin(
			goqu.T("locations").As("l1"),
			goqu.On(goqu.Ex{"t.from_location_id": goqu.I("l1.id")}),
		).
		LeftJoin(
			goqu.T("locations").As("l2"),
			goqu.On(goqu.Ex{"t.to_location_id": goqu.I("l2.id")}),
		)

	if conditions.HasConditions() {
		aliases := map[string]string{
			"from_location_id": "t.from_location_id",
			"to_location_id":   "t.to_location_id",
			"status":           "t.status",
		}

		query = query.Where(conditions.BuildConditions(aliases))
	}

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, fmt.Errorf("error building SQL query: %w", err)
	}

	log.Printf("Executing SQL query: %s with args: %v", sql, args)

	err = query.Executor().ScanStructs(&flatTransfers)
	if err != nil {
		return nil, fmt.Errorf("error executing SQL statement: %w", err)
	}

	return &flatTransfers, nil
}

func (r *transferRepository) UpdateTransferStatus(transferID int, status string) error {
	// TODO Transaction + remove transit status (do we really need this status?)
	query := r.Repo.GoquDBWrapper.
		Update("transfers").
		Set(goqu.Record{
			"status": status,
			// TODO "confirmed_at": goqu.L("NOW()"),
		}).
		Where(goqu.Ex{"id": transferID})

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to confirm transfer %d: %w", transferID, err)
	}

	return nil
}

func (r *transferRepository) MoveAssets(tx *goqu.TxDatabase, assets []int, locationID int, transitStatus string) error {
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

func (r *transferRepository) InsertTransferRecord(tx *goqu.TxDatabase, req models.TransferRequest) (int, error) {
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

func (r *transferRepository) InsertAssetsTransferRecord(tx *goqu.TxDatabase, transferID int, assets []int) error {
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

func (r *transferRepository) HasStockItemsInTransfer(tx *goqu.TxDatabase, transferID int) (bool, error) {
	var count int
	query := tx.From("non_serialized_transfers").
		Select(goqu.COUNT("id")).
		Where(goqu.Ex{"transfer_id": transferID})

	if _, err := query.Executor().ScanVal(&count); err != nil {
		return false, fmt.Errorf("failed to check stock items in transfer: %w", err)
	}

	return count > 0, nil
}

// TODO remodel to stock_id
func (r *transferRepository) InsertStockItemsTransferRecord(tx *goqu.TxDatabase, transferID int, stocks []models.StockItemRequest) error {
	var records []goqu.Record
	for _, stockItem := range stocks {
		records = append(records, goqu.Record{
			"transfer_id":      transferID,
			"item_category_id": goqu.L("(SELECT item_category_id FROM non_serialized_items WHERE id = ?)", stockItem.ID),
			"origin":           goqu.L("(SELECT origin FROM non_serialized_items WHERE id = ?)", stockItem.ID),
			"quantity":         stockItem.Quantity,
			"stock_id":         stockItem.ID,
		})
	}

	query := tx.Insert("non_serialized_transfers").Rows(records)

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to insert serialized asset transfers: %w", err)
	}

	return nil
}

func (r *transferRepository) RemoveStockItemsTransferRecords(tx *goqu.TxDatabase, transferID int) error {
	// Build the delete query
	query := tx.Delete("non_serialized_transfers").
		Where(goqu.Ex{"transfer_id": transferID})

	// Execute the query
	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to remove transfer records: %w", err)
	}

	return nil
}

func (r *transferRepository) GetTransferLocationById(tx *goqu.TxDatabase, transferID int) (int, error) {
	var locationId int
	_, err := tx.Select("to_location_id").
		From("transfers").
		Where(goqu.Ex{"id": transferID}).
		Executor().
		ScanVal(&locationId)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch to_location_id: %w", err)
	}
	return locationId, nil
}

func decreaseStockInTransfer(tx *goqu.TxDatabase, transferReq stocks.RemoveStockItemFromTransferRequest) error {
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
		return fmt.Errorf("insufficient stock for item_category_id %d at location ", transferReq.CategoryID)
	}

	return nil
}

func (r *transferRepository) InsertTransferUsers(tx *goqu.TxDatabase, transferID int, users []models.TransferUser) error {
	if len(users) == 0 {
		return nil
	}

	var records []goqu.Record
	for _, user := range users {
		records = append(records, goqu.Record{
			"transfer_id": transferID,
			"user_id":     user.UserID,
		})
	}

	query := tx.Insert("transfer_users").Rows(records)

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to insert transfer users: %w", err)
	}

	return nil
}

func (r *transferRepository) GetTransferUsers(transferID int) ([]models.User, error) {
	var users []models.User

	query := r.Repo.GoquDBWrapper.
		Select(
			"users.id",
			"users.username",
		).
		From("transfer_users").
		Join(goqu.T("users"), goqu.On(goqu.Ex{"transfer_users.user_id": goqu.I("users.id")})).
		Where(goqu.Ex{"transfer_id": transferID})

	err := query.Executor().ScanStructs(&users)
	if err != nil {
		return nil, fmt.Errorf("failed to get transfer users: %w", err)
	}

	return users, nil
}

func (r *transferRepository) GetTransfersByUserAndStatus(userID int, status string) ([]FlatTransfer, error) {
	var flatTransfers []FlatTransfer

	query := r.Repo.GoquDBWrapper.
		Select(
			goqu.I("t.id").As("transfer_id"),
			goqu.I("l1.id").As("from_location_id"),
			goqu.I("l1.name").As("from_location_name"),
			goqu.I("l1.pavilion").As("from_location_pavilion"),
			goqu.I("l2.id").As("to_location_id"),
			goqu.I("l2.name").As("to_location_name"),
			goqu.I("l2.pavilion").As("to_location_pavilion"),
			goqu.I("t.status").As("transfer_status"),
			goqu.I("t.transfer_date").As("transfer_date"),
		).
		From(goqu.T("transfers").As("t")).
		LeftJoin(
			goqu.T("locations").As("l1"),
			goqu.On(goqu.Ex{"t.from_location_id": goqu.I("l1.id")}),
		).
		LeftJoin(
			goqu.T("locations").As("l2"),
			goqu.On(goqu.Ex{"t.to_location_id": goqu.I("l2.id")}),
		).
		Join(
			goqu.T("transfer_users").As("tu"),
			goqu.On(goqu.Ex{"t.id": goqu.I("tu.transfer_id")}),
		).
		Where(goqu.Ex{"tu.user_id": userID})

	if status != "" {
		query = query.Where(goqu.Ex{"t.status": status})
	}

	err := query.Executor().ScanStructs(&flatTransfers)
	if err != nil {
		return nil, fmt.Errorf("error executing SQL statement: %w", err)
	}

	return flatTransfers, nil
}

func (r *transferRepository) UpdateDeliveryLocation(transferID int, latitude float64, longitude float64, timestamp time.Time) error {
	_, err := r.Repo.GoquDBWrapper.Update("transfers").
		Set(goqu.Record{
			"delivery_latitude":  latitude,
			"delivery_longitude": longitude,
			"delivery_timestamp": timestamp,
		}).
		Where(goqu.C("id").Eq(transferID)).
		Executor().
		Exec()

	return err
}

func (r *transferRepository) UpdateStockItemsTransferStatus(tx *goqu.TxDatabase, transferID int, status string) error {
	query := tx.Update("non_serialized_transfers").
		Set(goqu.Record{"status": status}).
		Where(goqu.Ex{"transfer_id": transferID})

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to update stock items transfer status: %w", err)
	}

	return nil
}

func (r *transferRepository) SetTransferUsers(transferID int, userIDs []int) error {
	return repository.WithTransaction(r.Repo.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		deleteQuery := tx.Delete("transfer_users").Where(goqu.Ex{"transfer_id": transferID})

		_, err := deleteQuery.Executor().Exec()
		if err != nil {
			return fmt.Errorf("failed to delete transfer users: %w", err)
		}

		var rows []goqu.Record
		for _, userID := range userIDs {
			rows = append(rows, goqu.Record{"transfer_id": transferID, "user_id": userID})
		}

		insertQuery := tx.Insert("transfer_users").Rows(rows)

		_, err = insertQuery.Executor().Exec()
		if err != nil {
			return fmt.Errorf("failed to set transfer user: %w", err)
		}

		return nil
	})
}
