package items

import (
	"fmt"
	"warehouse/internal/auditlog"
	"warehouse/internal/inventory/assets"
	"warehouse/internal/inventory/stocks"
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

func (s *ItemService) fetchItem(query RetrieveItemQuery) (interface{}, error) {
	switch query.CategoryType {
	case "asset":
		if query.ID == nil {
			return nil, fmt.Errorf("asset ID is required")
		}
		return s.ar.GetAsset(*query.ID)
	case "stock":
		if query.ID == nil {
			return nil, fmt.Errorf("stock ID is required")
		}
		return s.sr.GetStockItem(*query.ID)
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
		// Używamy warunków filtrowania jeśli są dostępne
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
			// Jeśli brak warunków, pobieramy wszystkie
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
		// Używamy warunków filtrowania jeśli są dostępne
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
			// Jeśli brak warunków, pobieramy wszystkie
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

	// Sprawdź czy są warunki filtrowania
	if conditions.HasConditions() {
		// Używamy warunków filtrowania dla obu typów
		queryBuilder := repository.NewQueryBuilder()
		queryBuilder.AddCondition("location_ids", conditions.LocationIDs)
		if conditions.CategoryID != nil {
			queryBuilder.AddCondition("category_id", *conditions.CategoryID)
		}
		if conditions.CategoryLabel != "" {
			queryBuilder.AddCondition("category_label", conditions.CategoryLabel)
		}

		// Pobierz zasoby z filtrowaniem
		assets, err := s.ar.GetAssetsBy(queryBuilder)
		if err != nil {
			return nil, err
		}
		for _, asset := range *assets {
			result = append(result, asset)
		}

		// Pobierz elementy magazynowe z filtrowaniem
		stocks, err := s.sr.GetStockItemsBy(queryBuilder)
		if err != nil {
			return nil, err
		}
		for _, stock := range *stocks {
			result = append(result, stock)
		}
	} else {
		// Jeśli brak warunków, pobieramy wszystkie elementy
		// Pobierz zasoby
		assets, err := s.ar.GetAssetList()
		if err != nil {
			return nil, err
		}
		for _, asset := range *assets {
			result = append(result, asset)
		}

		// Pobierz elementy magazynowe
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
