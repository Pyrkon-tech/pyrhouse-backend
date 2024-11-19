package repository

import (
	"fmt"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
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

func (r *Repository) getLocationAssets(locationID string) ([]models.Asset, error) {
	query := r.GoguDBWrapper.
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
	query := r.GoguDBWrapper.
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
