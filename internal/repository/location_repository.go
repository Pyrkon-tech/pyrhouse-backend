package repository

import (
	"fmt"
	"log"
	custom_error "warehouse/pkg/errors"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
	"github.com/lib/pq"
)

type LocationEquipment struct {
	Items              []models.Asset
	NonSerializedItems []models.StockItem
}

func (r *Repository) GetLocationEquipment(locationID string) (*models.LocationEquipment, error) {
	var locationEquipment models.LocationEquipment
	var err error

	locationEquipment.Assets, err = r.getLocationAssets(locationID)
	if err != nil {
		return nil, err
	}
	locationEquipment.StockItems, err = r.getLocationStock(locationID)
	if err != nil {
		return nil, err
	}

	return &locationEquipment, nil
}

func (r *Repository) PersistLocation(location *models.Location) error {
	query := r.goquDBWrapper.Insert("locations").
		Rows(goqu.Record{
			"name": location.Name,
		}).
		Returning("id")

	// TODO Value cannot be unique, there's a bug, no unique key in location table
	if _, err := query.Executor().ScanVal(&location.ID); err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return custom_error.WrapDBError("Duplicate serial number for asset", string(pqErr.Code))
			}
		}
		return fmt.Errorf("failed to insert location record: %w", err)
	}

	return nil
}

func (r *Repository) RemoveLocation(locationID string) error {
	result, err := r.goquDBWrapper.Delete("locations").Where(goqu.Ex{"id": locationID}).Executor().Exec()

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			return custom_error.WrapDBError("Duplicate serial number for asset", string(pqErr.Code))
		}
		log.Fatal("failed to delete location: ", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not retrieve rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no location found with id: %s", locationID)
	}

	return nil
}

func (r *Repository) getLocationAssets(locationID string) ([]models.Asset, error) {
	query := r.goquDBWrapper.
		From(goqu.T("items").As("i")).
		Select(
			"i.id",
			"i.item_serial",
			"i.item_category_id",
			"c.item_category",
			"c.label",
		)
	query = r.prepareQueryConditions(query, locationID)
	rows, err := query.Executor().Query()

	if err != nil {
		return nil, fmt.Errorf("error executing SQL statement: %w", err)
	}

	var assets []models.Asset
	for rows.Next() {
		var asset models.Asset
		if err := rows.Scan(
			&asset.ID,
			&asset.Serial,
			&asset.Category.ID,
			&asset.Category.Type,
			&asset.Category.Label,
		); err != nil {
			return nil, fmt.Errorf("unable fetch data: %w", err)
		}
		assets = append(assets, asset)
	}

	return assets, nil
}

func (r *Repository) getLocationStock(locationID string) ([]models.StockItem, error) {
	query := r.goquDBWrapper.
		From(goqu.T("non_serialized_items").As("i")).
		Select(
			"i.id",
			"i.quantity",
			"i.item_category_id",
			"c.item_category",
			"c.label",
		)
	query = r.prepareQueryConditions(query, locationID)
	rows, err := query.Executor().Query()

	if err != nil {
		return nil, fmt.Errorf("error executing SQL statement: %w", err)
	}

	var stockItems []models.StockItem
	for rows.Next() {
		var item models.StockItem
		if err := rows.Scan(
			&item.ID,
			&item.Quantity,
			&item.Category.ID,
			&item.Category.Type,
			&item.Category.Label,
		); err != nil {
			return nil, fmt.Errorf("unable fetch data: %w", err)
		}
		stockItems = append(stockItems, item)
	}

	return stockItems, nil
}

func (r *Repository) prepareQueryConditions(query *goqu.SelectDataset, locationID string) *goqu.SelectDataset {
	return query.
		LeftJoin(
			goqu.T("item_category").As("c"),
			goqu.On(goqu.Ex{"i.item_category_id": goqu.I("c.id")}),
		).
		Where(goqu.Ex{"i.location_id": locationID})
}
