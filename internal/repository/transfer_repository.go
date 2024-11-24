package repository

import (
	"fmt"
	"time"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
)

func (r *Repository) CanTransferNonSerializedItems(assets []models.UnserializedItemRequest, locationID int) (map[int]bool, error) {
	conditions := make([]goqu.Expression, 0, len(assets))
	for _, asset := range assets {
		conditions = append(conditions, goqu.And(
			goqu.C("item_category_id").Eq(asset.ItemCategoryID),
			goqu.C("location_id").Eq(locationID),
			goqu.C("quantity").Gte(asset.Quantity),
		))
	}

	sql, args, err := r.goquDBWrapper.From("non_serialized_items").
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

	err := withTransaction(r.goquDBWrapper, func(tx *goqu.TxDatabase) error {
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

type FlatTransfer struct {
	ID               int       `db:"transfer_id"`
	FromLocationID   int       `db:"from_location_id"`
	FromLocationName string    `db:"from_location_name"`
	ToLocationID     int       `db:"to_location_id"`
	ToLocationName   string    `db:"to_location_name"`
	TransferDate     time.Time `db:"transfer_date"`
	Status           string    `db:"transfer_status"`
}

func (r *Repository) GetTransfer(transferID int) (*models.Transfer, error) {
	var flatTransfer FlatTransfer

	query := r.goquDBWrapper.
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

	assets, err := r.fetchTransferAssets(transferID)
	if err != nil {
		return nil, err
	}
	stockItems, err := r.fetchTransferStock(transferID)
	if err != nil {
		return nil, err
	}

	transfer := models.Transfer{
		ID: flatTransfer.ID,
		FromLocation: models.Location{
			ID:   flatTransfer.FromLocationID,
			Name: flatTransfer.FromLocationName,
		},
		ToLocation: models.Location{
			ID:   flatTransfer.ToLocationID,
			Name: flatTransfer.ToLocationName,
		},
		TransferDate:         flatTransfer.TransferDate,
		Status:               flatTransfer.Status,
		AssetsCollection:     *assets,
		StockItemsCollection: *stockItems,
	}

	return &transfer, nil
}

func (r *Repository) ConfirmTransfer(transferID string, status string) error {
	// TODO Transaction + remove transit status (do we really need this status?)
	query := r.goquDBWrapper.
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
