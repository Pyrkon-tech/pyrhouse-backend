package assets

import (
	"database/sql"
	"fmt"
	"log"
	"warehouse/internal/repository"
	custom_error "warehouse/pkg/errors"
	"warehouse/pkg/metadata"
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
	query = query.
		Where(conditions.BuildConditions(aliases)).
		Order(goqu.I("i.id").Asc())

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

	record := goqu.Record{
		"location_id":      itemRequest.LocationId,
		"item_category_id": itemRequest.CategoryId,
		"status":           itemRequest.Status,
		"origin":           itemRequest.Origin,
	}

	if itemRequest.Serial != nil {
		record["item_serial"] = *itemRequest.Serial
	}

	query := r.repository.GoquDBWrapper.Insert("items").
		Rows(record).
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
	query := r.repository.GoquDBWrapper.Select("items.id").
		From(goqu.T("items")).
		Where(goqu.Ex{
			"items.id":          assetID,
			"items.location_id": models.DefaultEquipmentLocationID,
			"items.status":      goqu.Op{"in": []string{string(metadata.StatusInStock), string(metadata.StatusAvailable)}},
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

func (r *AssetsRepository) RemoveAsset(assetID int) (int, error) {
	var id int
	query := r.repository.GoquDBWrapper.
		Delete("items").
		Where(goqu.Ex{"id": assetID}).
		Returning("id")

	_, err := query.Executor().ScanVal(&id)

	if err != nil {
		log.Fatal("failed to delete asset category: ", err)
		return 0, err
	}

	return id, nil
}

func (r *AssetsRepository) UpdateAssetLocation(tx *goqu.TxDatabase, itemID int, locationID int) error {
	if tx == nil {
		return fmt.Errorf("transaction is required for UpdateAssetLocation")
	}

	// Najpierw sprawdzamy czy asset istnieje
	var exists bool
	_, err := tx.Select(goqu.L("1")).
		From("items").
		Where(goqu.Ex{"id": itemID}).
		Executor().
		ScanVal(&exists)

	if err != nil {
		return fmt.Errorf("failed to check if asset exists: %w", err)
	}

	if !exists {
		return fmt.Errorf("no asset found with id: %d", itemID)
	}

	// Aktualizujemy lokalizację
	_, err = tx.Update("items").
		Set(goqu.Record{"location_id": locationID}).
		Where(goqu.Ex{"id": itemID}).
		Executor().
		Exec()

	if err != nil {
		return fmt.Errorf("failed to update asset location: %w", err)
	}

	return nil
}

func (r *AssetsRepository) UpdateAssetStatusAndLocation(tx *goqu.TxDatabase, itemID int, locationID int, status metadata.Status) error {
	if tx == nil {
		return fmt.Errorf("transaction is required for UpdateAssetStatusAndLocation")
	}

	// Aktualizujemy status i lokalizację w jednym zapytaniu
	result, err := tx.Update("items").
		Set(goqu.Record{
			"location_id": locationID,
			"status":      string(status),
		}).
		Where(goqu.Ex{"id": itemID}).
		Executor().
		Exec()

	if err != nil {
		return fmt.Errorf("failed to update asset status and location: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no asset found with id: %d", itemID)
	}

	return nil
}

func (r *AssetsRepository) RemoveAssetFromTransfer(transferID int, itemID int, locationID int) error {
	return repository.WithTransaction(r.repository.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		// Najpierw sprawdzamy czy asset istnieje
		var count int
		_, err := tx.Select(goqu.COUNT("*")).
			From("items").
			Where(goqu.Ex{"id": itemID}).
			Executor().
			ScanVal(&count)

		if err != nil {
			return fmt.Errorf("failed to check if asset exists: %w", err)
		}

		if count == 0 {
			return fmt.Errorf("asset with id %d does not exist", itemID)
		}

		// Usuwamy z transferu
		result, err := tx.Delete("serialized_transfers").
			Where(goqu.Ex{
				"transfer_id": transferID,
				"item_id":     itemID,
			}).
			Executor().
			Exec()

		if err != nil {
			return fmt.Errorf("failed to remove asset from transfer: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return fmt.Errorf("no transfer record found for asset %d and transfer %d", itemID, transferID)
		}

		// Aktualizujemy status i lokalizację w jednym zapytaniu
		if err := r.UpdateAssetStatusAndLocation(tx, itemID, locationID, metadata.StatusAvailable); err != nil {
			return err
		}

		return nil
	})
}

func (r *AssetsRepository) GetTransferAssets(transferID int) (*[]models.Asset, error) {
	query := r.repository.GoquDBWrapper.
		Select(
			goqu.I("a.id").As("asset_id"),
			goqu.I("a.item_serial").As("item_serial"),
			"a.status",
			goqu.I("a.pyr_code").As("pyr_code"),
			goqu.I("a.origin").As("origin"),
			goqu.I("c.id").As("category_id"),
			goqu.I("c.item_category").As("category_type"),
			goqu.I("c.label").As("category_label"),
			goqu.I("c.pyr_id").As("category_pyr_id"),
			goqu.I("l.id").As("location_id"),
			goqu.I("l.name").As("location_name"),
			goqu.I("l.pavilion").As("location_pavilion"),
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

func (r *AssetsRepository) UpdateItemStatus(assetIDs []int, status metadata.Status, tx *goqu.TxDatabase) error {
	if len(assetIDs) == 0 {
		return nil
	}

	record := goqu.Record{"status": string(status)}
	condition := goqu.Ex{"id": assetIDs}

	var result sql.Result
	var err error

	if tx != nil {
		result, err = tx.Update("items").
			Set(record).
			Where(condition).
			Executor().
			Exec()
	} else {
		result, err = r.repository.GoquDBWrapper.Update("items").
			Set(record).
			Where(condition).
			Executor().
			Exec()
	}

	if err != nil {
		return fmt.Errorf("failed to update asset status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if int(rowsAffected) != len(assetIDs) {
		return fmt.Errorf("expected to update %d records, but updated %d", len(assetIDs), rowsAffected)
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
		goqu.I("l.pavilion").As("location_pavilion"),
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

func (r *AssetsRepository) CountAssetsInCategory(categoryID int) (int, error) {
	var count int
	query := r.repository.GoquDBWrapper.
		Select(goqu.COUNT("*")).
		From("items").
		Where(goqu.Ex{"item_category_id": categoryID})

	_, err := query.Executor().ScanVal(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count assets in category: %w", err)
	}

	return count, nil
}

func (r *AssetsRepository) GenerateUniquePyrCode(categoryID int, categoryPyrID string) (string, error) {
	var nextNumber int

	// Pobieramy największy numer dla danej kategorii
	query := r.repository.GoquDBWrapper.Select(
		goqu.L("COALESCE(MAX(CAST(REGEXP_REPLACE(pyr_code, '^PYR-" + categoryPyrID + "(\\d+)(-\\d+)?$', '\\1') AS INTEGER)), 0)"),
	).
		From("items").
		Where(goqu.L("pyr_code ~ ?", "^PYR-"+categoryPyrID+"\\d+(-\\d+)?$"))

	_, err := query.Executor().ScanVal(&nextNumber)
	if err != nil {
		return "", fmt.Errorf("failed to get next number: %w", err)
	}

	nextNumber++
	pyrCode := metadata.NewPyrCode(categoryPyrID, nextNumber)
	return pyrCode.GeneratePyrCode(), nil
}

func (r *AssetsRepository) UpdateAssetSerial(assetID int, serial string) error {
	query := r.repository.GoquDBWrapper.
		Update("items").
		Set(goqu.Record{"item_serial": serial}).
		Where(goqu.Ex{"id": assetID})

	result, err := query.Executor().Exec()
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			return custom_error.WrapDBError("Numer seryjny już istnieje", string(pqErr.Code))
		}
		return fmt.Errorf("nie udało się zaktualizować numeru seryjnego: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("nie udało się sprawdzić liczby zaktualizowanych wierszy: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("nie znaleziono zasobu o ID: %d", assetID)
	}

	return nil
}

func (r *AssetsRepository) GetAssetsForReport() ([]models.FlatAssetRecord, error) {
	query := r.getAssetQuery().
		Where(goqu.Ex{"c.category_type": "asset"}).
		Order(goqu.I("i.id").Asc())

	var flatAssets []models.FlatAssetRecord
	err := query.Executor().ScanStructs(&flatAssets)
	if err != nil {
		return nil, fmt.Errorf("nie udało się pobrać danych do raportu: %w", err)
	}

	return flatAssets, nil
}

func (r *AssetsRepository) GetStockForReport() ([]models.FlatStockRecord, error) {
	query := r.repository.GoquDBWrapper.Select(
		goqu.I("i.id").As("stock_id"),
		goqu.I("c.label").As("category_label"),
		goqu.I("i.origin").As("origin"),
		goqu.I("i.quantity").As("quantity"),
		goqu.I("l.name").As("location_name"),
	).
		From(goqu.T("non_serialized_items").As("i")).
		LeftJoin(
			goqu.T("item_category").As("c"),
			goqu.On(goqu.Ex{"i.item_category_id": goqu.I("c.id")}),
		).
		LeftJoin(
			goqu.T("locations").As("l"),
			goqu.On(goqu.Ex{"i.location_id": goqu.I("l.id")}),
		).
		Where(goqu.Ex{"c.category_type": "stock"}).
		Order(goqu.I("i.id").Asc())

	var flatStocks []models.FlatStockRecord
	err := query.Executor().ScanStructs(&flatStocks)
	if err != nil {
		return nil, fmt.Errorf("nie udało się pobrać danych do raportu: %w", err)
	}

	return flatStocks, nil
}
