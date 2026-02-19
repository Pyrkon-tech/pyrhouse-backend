package items

import (
	"fmt"
	"warehouse/internal/auditlog"
	"warehouse/internal/inventory/assets"
	"warehouse/internal/inventory/stocks"
	"warehouse/internal/models"
	"warehouse/internal/repository"
)

type ItemService struct {
	r        *repository.Repository
	sr       *stocks.StockRepository
	ar       *assets.AssetsRepository
	auditLog *auditlog.Auditlog
}

func NewItemService(r *repository.Repository, sr *stocks.StockRepository, ar *assets.AssetsRepository, al *auditlog.Auditlog) *ItemService {
	return &ItemService{
		r:        r,
		sr:       sr,
		ar:       ar,
		auditLog: al,
	}
}

type assetWithLogs struct {
	*models.Asset
	AssetLogs []models.AuditLog `json:"assetLogs"`
}

type stockWithLogs struct {
	*models.StockItem
	Logs []models.AuditLog `json:"logs"`
}

func (s *ItemService) fetchItem(query RetrieveItemQuery) (interface{}, error) {
	switch query.CategoryType {
	case "asset":
		if query.ID == nil {
			return nil, fmt.Errorf("asset ID is required")
		}
		asset, err := s.ar.GetAsset(*query.ID)
		if err != nil {
			return nil, err
		}
		logs, _ := s.auditLog.GetLogs(*query.ID, "asset")
		result := &assetWithLogs{Asset: asset}
		if logs != nil {
			result.AssetLogs = *logs
		}
		return result, nil
	case "stock":
		if query.ID == nil {
			return nil, fmt.Errorf("stock ID is required")
		}
		stock, err := s.sr.GetStockItem(*query.ID)
		if err != nil {
			return nil, err
		}
		logs, _ := s.auditLog.GetLogs(*query.ID, "stock")
		result := &stockWithLogs{StockItem: stock}
		if logs != nil {
			result.Logs = *logs
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported category type: %s", query.CategoryType)
	}
}

func (s *ItemService) fetchItemList(conditions RetrieveItemListQuery) ([]interface{}, error) {
	if conditions.CategoryType != "" {
		return s.fetchByCategory(conditions, conditions.CategoryType)
	}

	return s.fetchCombinedItems(conditions)
}

func (s *ItemService) fetchByCategory(conditions RetrieveItemListQuery, category string) ([]interface{}, error) {
	switch category {
	case "asset":
		// Use filter conditions if available
		if conditions.HasConditions() {
			queryBuilder := repository.NewQueryBuilder()
			queryBuilder.AddCondition("location_ids", conditions.LocationIDs)
			if conditions.CategoryID != nil {
				queryBuilder.AddCondition("category_id", *conditions.CategoryID)
			}
			if conditions.CategoryLabel != "" {
				queryBuilder.AddCondition("category_label", conditions.CategoryLabel)
			}

			assets, err := s.ar.GetAssetsBy(queryBuilder)
			if err != nil {
				return nil, err
			}
			result := make([]interface{}, len(*assets))
			for i, asset := range *assets {
				result[i] = asset
			}
			return result, nil
		} else {
			// If no conditions, fetch all
			assets, err := s.ar.GetAssetList()
			if err != nil {
				return nil, err
			}
			result := make([]interface{}, len(*assets))
			for i, asset := range *assets {
				result[i] = asset
			}
			return result, nil
		}
	case "stock":
		// Use filter conditions if available
		if conditions.HasConditions() {
			queryBuilder := repository.NewQueryBuilder()
			queryBuilder.AddCondition("location_ids", conditions.LocationIDs)
			if conditions.CategoryID != nil {
				queryBuilder.AddCondition("category_id", *conditions.CategoryID)
			}
			if conditions.CategoryLabel != "" {
				queryBuilder.AddCondition("category_label", conditions.CategoryLabel)
			}

			stocks, err := s.sr.GetStockItemsBy(queryBuilder)
			if err != nil {
				return nil, err
			}
			result := make([]interface{}, len(*stocks))
			for i, stock := range *stocks {
				result[i] = stock
			}
			return result, nil
		} else {
			// If no conditions, fetch all
			stocks, err := s.sr.GetStockItems()
			if err != nil {
				return nil, err
			}
			result := make([]interface{}, len(*stocks))
			for i, stock := range *stocks {
				result[i] = stock
			}
			return result, nil
		}
	default:
		return nil, fmt.Errorf("unsupported category type: %s", category)
	}
}

func (s *ItemService) fetchCombinedItems(conditions RetrieveItemListQuery) ([]interface{}, error) {
	var result []interface{}

	// Check if there are filter conditions
	if conditions.HasConditions() {
		// Use filter conditions for both types
		queryBuilder := repository.NewQueryBuilder()
		queryBuilder.AddCondition("location_ids", conditions.LocationIDs)
		if conditions.CategoryID != nil {
			queryBuilder.AddCondition("category_id", *conditions.CategoryID)
		}
		if conditions.CategoryLabel != "" {
			queryBuilder.AddCondition("category_label", conditions.CategoryLabel)
		}

		// Fetch assets with filtering
		assets, err := s.ar.GetAssetsBy(queryBuilder)
		if err != nil {
			return nil, err
		}
		for _, asset := range *assets {
			result = append(result, asset)
		}

		// Fetch stock items with filtering
		stocks, err := s.sr.GetStockItemsBy(queryBuilder)
		if err != nil {
			return nil, err
		}
		for _, stock := range *stocks {
			result = append(result, stock)
		}
	} else {
		// If no conditions, fetch all items
		// Fetch assets
		assets, err := s.ar.GetAssetList()
		if err != nil {
			return nil, err
		}
		for _, asset := range *assets {
			result = append(result, asset)
		}

		// Fetch stock items
		stocks, err := s.sr.GetStockItems()
		if err != nil {
			return nil, err
		}
		for _, stock := range *stocks {
			result = append(result, stock)
		}
	}

	return result, nil
}
