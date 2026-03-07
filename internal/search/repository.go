package search

import (
	"fmt"
	"warehouse/internal/models"
	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
)

type Repository struct {
	repo *repository.Repository
}

func NewRepository(repo *repository.Repository) *Repository {
	return &Repository{repo: repo}
}

type SearchResult struct {
	Assets []models.Asset     `json:"assets"`
	Stocks []models.StockItem `json:"stocks"`
}

func (r *Repository) Search(query string) (*SearchResult, error) {
	pattern := "%" + query + "%"

	assets, err := r.searchAssets(pattern)
	if err != nil {
		return nil, fmt.Errorf("asset search failed: %w", err)
	}

	stocks, err := r.searchStocks(pattern)
	if err != nil {
		return nil, fmt.Errorf("stock search failed: %w", err)
	}

	return &SearchResult{
		Assets: assets,
		Stocks: stocks,
	}, nil
}

func (r *Repository) searchAssets(pattern string) ([]models.Asset, error) {
	query := r.repo.GoquDBWrapper.
		Select(
			goqu.I("i.id").As("asset_id"),
			"i.status",
			goqu.I("i.item_serial").As("item_serial"),
			goqu.I("i.pyr_code").As("pyr_code"),
			goqu.L("CASE WHEN i.origin_suffix IS NOT NULL THEN o.slug || '-' || i.origin_suffix ELSE o.slug END").As("origin"),
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
		LeftJoin(goqu.T("origins").As("o"), goqu.On(goqu.Ex{"i.origin_id": goqu.I("o.id")})).
		LeftJoin(goqu.T("item_category").As("c"), goqu.On(goqu.Ex{"i.item_category_id": goqu.I("c.id")})).
		LeftJoin(goqu.T("locations").As("l"), goqu.On(goqu.Ex{"i.location_id": goqu.I("l.id")})).
		Where(goqu.Or(
			goqu.I("i.pyr_code").ILike(pattern),
			goqu.I("i.item_serial").ILike(pattern),
			goqu.I("c.label").ILike(pattern),
			goqu.I("c.item_category").ILike(pattern),
			goqu.I("l.name").ILike(pattern),
			goqu.I("o.slug").ILike(pattern),
		)).
		Limit(20)

	var flatAssets []models.FlatAssetRecord
	if err := query.Executor().ScanStructs(&flatAssets); err != nil {
		return nil, fmt.Errorf("error searching assets: %w", err)
	}

	assets := make([]models.Asset, len(flatAssets))
	for i, fa := range flatAssets {
		assets[i] = fa.TransformToAsset()
	}
	return assets, nil
}

func (r *Repository) searchStocks(pattern string) ([]models.StockItem, error) {
	query := r.repo.GoquDBWrapper.
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
		LeftJoin(goqu.T("origins").As("o"), goqu.On(goqu.Ex{"s.origin_id": goqu.I("o.id")})).
		LeftJoin(goqu.T("item_category").As("c"), goqu.On(goqu.Ex{"s.item_category_id": goqu.I("c.id")})).
		LeftJoin(goqu.T("locations").As("l"), goqu.On(goqu.Ex{"s.location_id": goqu.I("l.id")})).
		Where(goqu.Or(
			goqu.I("c.label").ILike(pattern),
			goqu.I("c.item_category").ILike(pattern),
			goqu.I("l.name").ILike(pattern),
			goqu.I("o.slug").ILike(pattern),
		)).
		Where(goqu.C("quantity").Gt(0)).
		Limit(20)

	var flatStocks []models.FlatStockRecord
	if err := query.Executor().ScanStructs(&flatStocks); err != nil {
		return nil, fmt.Errorf("error searching stocks: %w", err)
	}

	stocks := make([]models.StockItem, len(flatStocks))
	for i, fs := range flatStocks {
		stocks[i] = models.StockItem{
			ID:       fs.ID,
			Quantity: fs.Quantity,
			Origin:   fs.Origin,
			Category: models.ItemCategory{
				ID:    fs.CategoryID,
				Name:  fs.CategoryType,
				Label: fs.CategoryLabel,
				Type:  fs.CategoryEquipmentType,
			},
			Location: models.Location{
				ID:   fs.LocationID,
				Name: fs.LocationName,
			},
		}
	}
	return stocks, nil
}
