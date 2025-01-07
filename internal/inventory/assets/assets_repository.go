package assets

import (
	"fmt"
	"log"
	"warehouse/internal/repository"
	custom_error "warehouse/pkg/errors"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
	"github.com/lib/pq"
)

type AssetsRepository struct {
	repository *repository.Repository
}

func NewRepository(r *repository.Repository) *AssetsRepository {
	return &AssetsRepository{
		repository: r,
	}
}

func (r *AssetsRepository) GetAsset(id int) (*models.Asset, error) {
	return r.fetchFlatAssetByCondition(goqu.Ex{"i.id": id})
}

func (r *AssetsRepository) GetAssetList() (*[]models.Asset, error) {
	query := r.getAssetQuery()

	var flatAssets []models.FlatAssetRecord
	err := query.Executor().ScanStructs(&flatAssets)

	if err != nil {
		return nil, fmt.Errorf("unable to select assets from database: %s", err.Error())
	}

	var assets []models.Asset
	for _, flatAsset := range flatAssets {
		asset := flatAsset.TransformToAsset()
		assets = append(assets, asset)
	}

	return &assets, nil
}

func (r *AssetsRepository) GetAssetsBy(conditions repository.QueryBuilder) (*[]models.Asset, error) {
	aliases := map[string]string{
		"location_ids":   "i.location_id",
		"category_id":    "i.item_category_id",
		"category_label": "c.label",
	}

	query := r.getAssetQuery()
	query = query.Where(conditions.BuildConditions(aliases))

	var flatAssets []models.FlatAssetRecord
	err := query.Executor().ScanStructs(&flatAssets)

	if err != nil {
		return nil, fmt.Errorf("unable to select assets from database: %s", err.Error())
	}

	var assets []models.Asset
	for _, flatAsset := range flatAssets {
		asset := flatAsset.TransformToAsset()
		assets = append(assets, asset)
	}

	return &assets, nil
}

func (r *AssetsRepository) FindItemByPyrCode(pyrCode string) (*models.Asset, error) {
	return r.fetchFlatAssetByCondition(goqu.Ex{"i.pyr_code": pyrCode})
}

func (r *AssetsRepository) HasRelatedItems(categoryID string) bool {
	query := `SELECT COUNT(*) FROM items WHERE item_category_id = $1`
	var count int
	err := r.repository.DB.QueryRow(query, categoryID).Scan(&count)
	if err != nil {
		log.Fatal("failed to check related assets: ", err)

		return false
	}
	return count > 0
}

func (r *AssetsRepository) HasItemsInLocation(assetIDs []int, fromLocationId int) (bool, error) {
	sql, args, err := r.repository.GoquDBWrapper.Select(goqu.COUNT("id")).From("items").Where(goqu.Ex{
		"location_id": fromLocationId,
		"id":          assetIDs,
	}).ToSQL()

	if err != nil {
		log.Fatalf("Failed to build query: %v", err)
	}

	var count int
	err = r.repository.DB.QueryRow(sql, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to execute query: %w", err)
	}

	return count == len(assetIDs), nil
}

func (r *AssetsRepository) PersistItem(itemRequest models.ItemRequest) (*models.Asset, error) {
	var assetID int

	query := r.repository.GoquDBWrapper.Insert("items").
		Rows(goqu.Record{
			"item_serial":      itemRequest.Serial,
			"location_id":      itemRequest.LocationId,
			"item_category_id": itemRequest.CategoryId,
			"status":           itemRequest.Status,
			"origin":           itemRequest.Origin,
		}).
		Returning("id")

	if _, err := query.Executor().ScanVal(&assetID); err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			return nil, custom_error.WrapDBError("Duplicate serial number for asset", string(pqErr.Code))
		}
		return nil, fmt.Errorf("failed to insert asset record: %w", err)
	}

	asset, err := r.GetAsset(assetID)

	if err != nil {
		return nil, err
	}

	return asset, nil
}

func (r *AssetsRepository) CanRemoveAsset(assetID int) (bool, error) {
	var id int
	query := r.repository.GoquDBWrapper.Select("i.id").
		From(goqu.T("items").As("i")).
		Where(goqu.Ex{
			"i.id":          assetID,
			"i.location_id": models.DefaultEquipmentLocationID,
			"i.status":      models.AssetStatusInStock,
		}).
		Where(goqu.L("NOT EXISTS (?)",
			r.repository.GoquDBWrapper.From(goqu.T("serialized_transfers").As("st")).
				Select(goqu.L("1")).
				Where(goqu.Ex{
					"st.item_id": assetID,
				}),
		))
	result, err := query.Executor().ScanVal(&id)

	if err != nil {
		return false, fmt.Errorf("unable to execute sql: %w", err)
	}

	return result, nil
}

func (r *AssetsRepository) RemoveAsset(assetID int) (string, error) {
	var assetSerial string
	query := r.repository.GoquDBWrapper.
		Delete("items").
		Where(goqu.Ex{"id": assetID}).
		Returning("item_serial")

	_, err := query.Executor().ScanVal(&assetSerial)

	if err != nil {
		log.Fatal("failed to delete asset category: ", err)
		return "", err
	}

	return assetSerial, nil
}

func (r *AssetsRepository) RemoveAssetFromTransfer(transferID int, itemID int, locationID int) error {
	err := repository.WithTransaction(r.repository.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		var err error
		_, err = tx.Delete("serialized_transfers").
			Where(goqu.Ex{
				"transfer_id": transferID,
				"item_id":     itemID,
			}).
			Executor().
			Exec()

		if err != nil {
			return fmt.Errorf("failed to remove asset from transfer %d: %w", transferID, err)
		}

		_, err = tx.Update("items").
			Set(goqu.Record{"location_id": locationID}).
			Where(goqu.Ex{"id": itemID}).
			Executor().
			Exec()

		if err != nil {
			return fmt.Errorf("failed to remove asset from transfer, unable to update location %d: %w", transferID, err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (r *AssetsRepository) GetTransferAssets(transferID int) (*[]models.Asset, error) {
	query := r.repository.GoquDBWrapper.
		Select(
			goqu.I("a.id").As("asset_id"),
			goqu.I("a.item_serial").As("item_serial"),
			"a.status",
			goqu.I("a.pyr_code").As("pyr_code"),
			goqu.I("c.id").As("category_id"),
			goqu.I("c.item_category").As("category_type"),
			goqu.I("c.label").As("category_label"),
			goqu.I("c.pyr_id").As("category_pyr_id"),
			goqu.I("l.id").As("location_id"),
			goqu.I("l.name").As("location_name"),
		).
		From(goqu.T("serialized_transfers").As("ta")).
		LeftJoin(
			goqu.T("items").As("a"),
			goqu.On(goqu.Ex{"ta.item_id": goqu.I("a.id")}),
		).
		LeftJoin(
			goqu.T("item_category").As("c"),
			goqu.On(goqu.Ex{"a.item_category_id": goqu.I("c.id")}),
		).
		LeftJoin(
			goqu.T("locations").As("l"),
			goqu.On(goqu.Ex{"a.location_id": goqu.I("l.id")}),
		).
		Where(goqu.Ex{"ta.transfer_id": transferID})

	var flatAssets []models.FlatAssetRecord

	err := query.Executor().ScanStructs(&flatAssets)

	if err != nil {
		return nil, fmt.Errorf("error executing SQL statement for assets: %w", err)
	}

	var assets []models.Asset
	for _, flatAsset := range flatAssets {
		asset := flatAsset.TransformToAsset()
		assets = append(assets, asset)
	}

	return &assets, nil
}

func (r *AssetsRepository) UpdatePyrCode(assetID int, pyrCode string) error {
	record := goqu.Record{"pyr_code": pyrCode}
	condition := goqu.Ex{"id": assetID}
	err := r.updateAsset(record, condition)
	if err != nil {
		return fmt.Errorf("failed to update asset pyrcode: %w", err)
	}

	return nil
}

func (r *AssetsRepository) UpdateItemStatus(assetIDs []int, status string) error {
	record := goqu.Record{"status": status}
	condition := goqu.Ex{"id": assetIDs}
	err := r.updateAsset(record, condition)
	if err != nil {
		return fmt.Errorf("failed to update asset status: %w", err)
	}

	return nil
}

func (r *AssetsRepository) updateAsset(record goqu.Record, condition goqu.Expression) error {
	query := r.repository.GoquDBWrapper.
		Update("items").
		Set(record).
		Where(condition)

	_, err := query.Executor().Exec()

	return err
}

func (r *AssetsRepository) fetchFlatAssetByCondition(condition goqu.Expression) (*models.Asset, error) {
	query := r.getAssetQuery().Where(condition)

	var flatAsset models.FlatAssetRecord
	_, err := query.Executor().ScanStruct(&flatAsset)

	if err != nil {
		return nil, fmt.Errorf("unable to select asset from database: %s", err.Error())
	}
	asset := flatAsset.TransformToAsset()

	return &asset, nil
}

func (r *AssetsRepository) getAssetQuery() *goqu.SelectDataset {
	query := r.repository.GoquDBWrapper.Select(
		goqu.I("i.id").As("asset_id"),
		"i.status",
		goqu.I("i.item_serial").As("item_serial"),
		goqu.I("i.pyr_code").As("pyr_code"),
		goqu.I("i.origin").As("origin"),
		goqu.I("c.id").As("category_id"),
		goqu.I("c.item_category").As("category_type"),
		goqu.I("c.label").As("category_label"),
		goqu.I("c.pyr_id").As("category_pyr_id"),
		goqu.I("c.category_type").As("category_equipment_type"),
		goqu.I("l.id").As("location_id"),
		goqu.I("l.name").As("location_name"),
	).
		From(goqu.T("items").As("i")).
		LeftJoin(
			goqu.T("item_category").As("c"),
			goqu.On(goqu.Ex{"i.item_category_id": goqu.I("c.id")}),
		).
		LeftJoin(
			goqu.T("locations").As("l"),
			goqu.On(goqu.Ex{"i.location_id": goqu.I("l.id")}),
		)
	return query
}
