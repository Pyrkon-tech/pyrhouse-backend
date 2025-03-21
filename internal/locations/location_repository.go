package locations

import (
	"fmt"
	"log"
	"warehouse/internal/repository"
	custom_error "warehouse/pkg/errors"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
	"github.com/lib/pq"
)

type LocationRepository struct {
	Repository *repository.Repository
}

type LocationEquipment struct {
	Items              []models.Asset
	NonSerializedItems []models.StockItem
}

func NewLocationRepository(r *repository.Repository) *LocationRepository {
	return &LocationRepository{Repository: r}
}

func (r *LocationRepository) GetLocations() (*[]models.Location, error) {
	var locations = []models.Location{}
	query := r.Repository.GoquDBWrapper.Select("id", "name", "details").From("locations")
	if err := query.Executor().ScanStructs(&locations); err != nil {
		return nil, fmt.Errorf("unable to execute SQL: %w", err)
	}

	return &locations, nil
}

func (r *LocationRepository) GetLocationEquipment(locationID string) (*models.LocationEquipment, error) {
	var locationEquipment models.LocationEquipment
	var err error
	// TODO error handling
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

func (r *LocationRepository) PersistLocation(location *models.Location) error {
	query := r.Repository.GoquDBWrapper.Insert("locations").
		Rows(goqu.Record{
			"name":    location.Name,
			"details": location.Details,
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

func (r *LocationRepository) UpdateLocation(locationID string, req UpdateLocationRequest) (models.Location, error) {
	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Details != nil {
		updates["details"] = *req.Details
	}
	if len(updates) == 0 {
		return models.Location{}, fmt.Errorf("no fields to update")
	}

	query := r.Repository.GoquDBWrapper.
		Update("locations").
		Set(updates).
		Where(goqu.Ex{"id": locationID}).
		Returning("id", "name", "details")

	var loc models.Location

	_, err := query.Executor().ScanStruct(&loc)
	if err != nil {
		return models.Location{}, fmt.Errorf("failed to update location: %w", err)
	}

	return loc, nil
}

func (r *LocationRepository) RemoveLocation(locationID string) error {
	result, err := r.Repository.GoquDBWrapper.Delete("locations").Where(goqu.Ex{"id": locationID}).Executor().Exec()

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

// TODO maybe remove?
func (r *LocationRepository) getLocationAssets(locationID string) ([]models.Asset, error) {
	query := r.Repository.GoquDBWrapper.
		From(goqu.T("items").As("i")).
		Select(
			"i.id",
			"i.item_serial",
			"i.status",
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
			&asset.Status,
			&asset.Category.ID,
			&asset.Category.Name,
			&asset.Category.Label,
		); err != nil {
			return nil, fmt.Errorf("unable fetch data: %w", err)
		}
		assets = append(assets, asset)
	}

	return assets, nil
}

func (r *LocationRepository) getLocationStock(locationID string) ([]models.StockItem, error) {
	query := r.Repository.GoquDBWrapper.
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
			&item.Category.Name,
			&item.Category.Label,
		); err != nil {
			return nil, fmt.Errorf("unable fetch data: %w", err)
		}
		stockItems = append(stockItems, item)
	}

	return stockItems, nil
}

func (r *LocationRepository) prepareQueryConditions(query *goqu.SelectDataset, locationID string) *goqu.SelectDataset {
	return query.
		LeftJoin(
			goqu.T("item_category").As("c"),
			goqu.On(goqu.Ex{"i.item_category_id": goqu.I("c.id")}),
		).
		Where(goqu.Ex{"i.location_id": locationID})
}

func (r *LocationRepository) SearchLocationItems(locationID string, searchQuery string) ([]models.Asset, error) {
	query := r.Repository.GoquDBWrapper.
		From(goqu.T("items").As("i")).
		Select(
			"i.id",
			"i.item_serial",
			"i.status",
			"i.pyr_code",
			"i.origin",
			"i.item_category_id",
			"c.item_category",
			"c.label",
		).
		LeftJoin(
			goqu.T("item_category").As("c"),
			goqu.On(goqu.Ex{"i.item_category_id": goqu.I("c.id")}),
		).
		Where(goqu.Ex{
			"i.location_id": locationID,
		}).
		Where(goqu.Or(
			goqu.I("i.item_serial").ILike("%"+searchQuery+"%"),
			goqu.I("c.item_category").ILike("%"+searchQuery+"%"),
			goqu.I("c.label").ILike("%"+searchQuery+"%"),
			goqu.I("i.pyr_code").ILike("%"+searchQuery+"%"),
		))

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
			&asset.Status,
			&asset.PyrCode,
			&asset.Origin,
			&asset.Category.ID,
			&asset.Category.Name,
			&asset.Category.Label,
		); err != nil {
			return nil, fmt.Errorf("unable fetch data: %w", err)
		}
		assets = append(assets, asset)
	}

	return assets, nil
}
