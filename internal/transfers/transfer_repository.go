package transfers

import (
	"fmt"
	"time"
	"warehouse/internal/repository"
	"warehouse/internal/stocks"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
)

type TransferRepository interface {
	CanTransferNonSerializedItems(assets []models.StockItemRequest, locationID int) (map[int]bool, error)
	ConfirmTransfer(transferID int, status string) error
	GetTransferRow(transferID int) (*FlatTransfer, error)
	GetTransferRows() (*[]FlatTransfer, error)
	InsertTransferRecord(tx *goqu.TxDatabase, req models.TransferRequest) (int, error)
	GetTransferLocationById(tx *goqu.TxDatabase, transferID int) (int, error)
	InsertSerializedItemTransferRecord(tx *goqu.TxDatabase, transferID int, assets []int) error
	MoveSerializedItems(tx *goqu.TxDatabase, assets []int, locationID int, transitStatus string) error
	InsertNonSerializedItemTransferRecord(tx *goqu.TxDatabase, transferID int, unserializedItems []models.StockItemRequest) error
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
	ID               int       `db:"transfer_id"`
	FromLocationID   int       `db:"from_location_id"`
	FromLocationName string    `db:"from_location_name"`
	ToLocationID     int       `db:"to_location_id"`
	ToLocationName   string    `db:"to_location_name"`
	TransferDate     time.Time `db:"transfer_date"`
	Status           string    `db:"transfer_status"`
}

// TODO Service method
func (r *transferRepository) GetTransferRow(transferID int) (*FlatTransfer, error) {
	var flatTransfer FlatTransfer

	query := r.Repo.GoquDBWrapper.
		Select(
			goqu.I("t.id").As("transfer_id"),
			goqu.I("l1.id").As("from_location_id"),
			goqu.I("l1.name").As("from_location_name"),
			goqu.I("l2.id").As("to_location_id"),
			goqu.I("l2.name").As("to_location_name"),
			goqu.I("t.status").As("transfer_status"),
			goqu.I("t.transfer_date").As("transfer_date"),
			//TODO goqu.I("t.receiver").As("transfer_receiver"),
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
	_, err := query.Executor().ScanStruct(&flatTransfer)
	if err != nil {
		return nil, fmt.Errorf("error executing SQL statement: %w", err)
	}

	return &flatTransfer, nil
}

func (r *transferRepository) GetTransferRows() (*[]FlatTransfer, error) {
	var flatTransfers []FlatTransfer

	query := r.Repo.GoquDBWrapper.
		Select(
			goqu.I("t.id").As("transfer_id"),
			goqu.I("l1.id").As("from_location_id"),
			goqu.I("l1.name").As("from_location_name"),
			goqu.I("l2.id").As("to_location_id"),
			goqu.I("l2.name").As("to_location_name"),
			goqu.I("t.status").As("transfer_status"),
			goqu.I("t.transfer_date").As("transfer_date"),
			//TODO goqu.I("t.receiver").As("transfer_receiver"),
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
	err := query.Executor().ScanStructs(&flatTransfers)

	if err != nil {
		return nil, fmt.Errorf("error executing SQL statement: %w", err)
	}

	return &flatTransfers, nil
}

func (r *transferRepository) ConfirmTransfer(transferID int, status string) error {
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

func (r *transferRepository) MoveSerializedItems(tx *goqu.TxDatabase, assets []int, locationID int, transitStatus string) error {
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

func (r *transferRepository) InsertSerializedItemTransferRecord(tx *goqu.TxDatabase, transferID int, assets []int) error {
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

// TODO remodel to stock_id
func (r *transferRepository) InsertNonSerializedItemTransferRecord(tx *goqu.TxDatabase, transferID int, stocks []models.StockItemRequest) error {
	var records []goqu.Record
	for _, stockItem := range stocks {
		records = append(records, goqu.Record{
			"transfer_id":      transferID,
			"item_category_id": goqu.L("(SELECT item_category_id FROM non_serialized_items WHERE id = ?)", stockItem.ID),
			"stock_id":         stockItem.ID,
			"quantity":         stockItem.Quantity,
		})
	}

	query := tx.Insert("non_serialized_transfers").Rows(records)

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to insert serialized asset transfers: %w", err)
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
		return fmt.Errorf("insufficient stock for item_category_id %d at location %d", transferReq.CategoryID, transferReq.LocationID)
	}

	return nil
}
