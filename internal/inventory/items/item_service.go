package items

import (
	"fmt"
	"warehouse/internal/auditlog"
	"warehouse/internal/inventory/assets"
	"warehouse/internal/inventory/stocks"
	"warehouse/internal/repository"
)

type ItemService struct {
	r                  *repository.Repository
	sr                 *stocks.StockRepository
	ar                 *assets.AssetsRepository
	auditlogRepository *auditlog.AuditLogRepository
}

func (s *ItemService) fetchItem(query retrieveItemQuery) (interface{}, error) {
	switch query.CategoryType {
	case "asset":
		asset, err := s.ar.GetAsset(*query.ID)
		if err != nil {
			return nil, err
		}
		assetLogs, err := s.auditlogRepository.GetResourceLog(*query.ID, query.CategoryType)
		if err != nil {
			return nil, err
		}
		item := map[string]interface{}{
			"asset":     asset,
			"assetLogs": assetLogs,
		}

		return item, nil
	case "stock":
		stock, err := s.sr.GetStockItem(*query.ID)
		if err != nil {
			return nil, err
		}
		stockLogs, err := s.auditlogRepository.GetResourceLog(*query.ID, query.CategoryType)
		if err != nil {
			return nil, err
		}
		item := map[string]interface{}{
			"stock":     stock,
			"assetLogs": stockLogs,
		}

		return item, nil
	default:
		return nil, fmt.Errorf("invalid item type provided")
	}
}

func (s *ItemService) fetchItemList(conditions retrieveItemListQuery) ([]interface{}, error) {
	switch conditions.CategoryType {
	case "asset":
		return s.fetchByCategory(conditions, "asset")
	case "stock":
		return s.fetchByCategory(conditions, "stock")
	default:
		return s.fetchCombinedItems(conditions)
	}
}

func (s *ItemService) fetchByCategory(conditions retrieveItemListQuery, category string) ([]interface{}, error) {
	var items []interface{}
	var err error

	switch category {
	case "asset":
		assets, fetchErr := s.ar.GetAssetsBy(&conditions)
		err = fetchErr
		for _, asset := range *assets {
			items = append(items, asset)
		}
	case "stock":
		stocks, fetchErr := s.sr.GetStockItemsBy(&conditions)
		err = fetchErr
		for _, stock := range *stocks {
			items = append(items, stock)
		}
	}

	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *ItemService) fetchCombinedItems(conditions retrieveItemListQuery) ([]interface{}, error) {
	assetFetcher := func() ([]interface{}, error) {
		assets, err := s.ar.GetAssetsBy(&conditions)
		if err != nil {
			return nil, err
		}
		var items []interface{}
		for _, asset := range *assets {
			items = append(items, asset)
		}
		return items, nil
	}

	stockFetcher := func() ([]interface{}, error) {
		stocks, err := s.sr.GetStockItemsBy(&conditions)
		if err != nil {
			return nil, err
		}
		var items []interface{}
		for _, stock := range *stocks {
			items = append(items, stock)
		}
		return items, nil
	}

	return parallelFetch(assetFetcher, stockFetcher)
}

func parallelFetch(fetchers ...func() ([]interface{}, error)) ([]interface{}, error) {
	results := make(chan []interface{}, len(fetchers))
	errors := make(chan error, len(fetchers))

	for _, fetcher := range fetchers {
		go func(f func() ([]interface{}, error)) {
			res, err := f()
			if err != nil {
				errors <- err
				return
			}
			results <- res
		}(fetcher)
	}

	var combined []interface{}
	for i := 0; i < len(fetchers); i++ {
		select {
		case err := <-errors:
			return nil, err
		case res := <-results:
			combined = append(combined, res...)
		}
	}

	return combined, nil
}
