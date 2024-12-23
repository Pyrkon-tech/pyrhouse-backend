package items

import (
	"warehouse/internal/repository"
	"warehouse/internal/stocks"
	"warehouse/pkg/models"
)

type ItemService struct {
	r  *repository.Repository
	sr *stocks.StockRepository
}

func (s *ItemService) fetchItems(conditions fetchItemsQuery) (*[]interface{}, error) {
	switch conditions.CategoryType {
	case "asset":
		assets, err := s.r.GetAssetsBy(&conditions)
		if err != nil {
			return nil, err
		}
		return s.combinedItems(assets, nil), nil
	case "stock":
		stocks, err := s.sr.GetStockItemsBy(&conditions)

		if err != nil {
			return nil, err
		}
		return s.combinedItems(nil, stocks), nil
	default:
		return s.fetchCombinedItems(conditions)
	}
}

func (s *ItemService) fetchCombinedItems(conditions fetchItemsQuery) (*[]interface{}, error) {
	assetChannel := make(chan *[]models.Asset, 1)
	stockChannel := make(chan *[]models.StockItem, 1)
	errChannel := make(chan error, 2)

	go func() {
		assets, err := s.r.GetAssetsBy(&conditions)

		if err != nil {
			errChannel <- err
			return
		}
		assetChannel <- assets
	}()

	go func() {
		stocks, err := s.sr.GetStockItemsBy(&conditions)

		if err != nil {
			errChannel <- err
			return
		}
		stockChannel <- stocks
	}()

	var assets *[]models.Asset
	var stocks *[]models.StockItem

	for i := 0; i < 2; i++ {
		select {
		case result := <-assetChannel:
			assets = result
		case result := <-stockChannel:
			stocks = result
		case err := <-errChannel:
			return nil, err
		}
	}

	return s.combinedItems(assets, stocks), nil
}

func (s *ItemService) combinedItems(assets *[]models.Asset, stocks *[]models.StockItem) *[]interface{} {
	var combinedItems []interface{}

	if assets != nil {
		for _, asset := range *assets {
			combinedItems = append(combinedItems, asset)
		}
	}

	if stocks != nil {
		for _, stock := range *stocks {
			combinedItems = append(combinedItems, stock)
		}
	}

	return &combinedItems
}
