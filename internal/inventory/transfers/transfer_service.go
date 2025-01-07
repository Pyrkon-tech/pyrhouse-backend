package transfers

import (
	"fmt"
	"warehouse/internal/inventory/assets"
	"warehouse/internal/inventory/stocks"
	"warehouse/internal/repository"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
)

type TransferService struct {
	r         *repository.Repository
	tr        TransferRepository
	ar        *assets.AssetsRepository
	stockRepo *stocks.StockRepository
}

func (s *TransferService) PerformTransfer(req models.TransferRequest, transitStatus string) (int, error) {
	var transferID int

	err := repository.WithTransaction(s.r.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		var err error
		if transferID, err = s.tr.InsertTransferRecord(tx, req); err != nil {
			return fmt.Errorf("failed to insert transfer record: %w", err)
		}

		if err = s.handleSerializedItems(tx, transferID, req.AssetItemCollection, req.LocationID, transitStatus); err != nil {
			return err
		}

		if err = s.handleNonSerializedItems(tx, transferID, req.StockItemCollection, req.LocationID, req.FromLocationID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return transferID, nil
}

func (s *TransferService) GetTransfer(transferID int) (*models.Transfer, error) {
	flatTransfer, err := s.tr.GetTransferRow(transferID)
	if err != nil {
		return nil, err
	}
	assets, err := s.ar.GetTransferAssets(transferID)
	if err != nil {
		return nil, err
	}
	stockItems, err := s.stockRepo.GetStockItemsByTransfer(transferID)
	if err != nil {
		return nil, err
	}

	transfer := models.Transfer{
		ID: flatTransfer.ID,
		FromLocation: models.Location{
			ID:   flatTransfer.FromLocationID,
			Name: flatTransfer.FromLocationName,
		},
		ToLocation: models.Location{
			ID:   flatTransfer.ToLocationID,
			Name: flatTransfer.ToLocationName,
		},
		AssetsCollection:     *assets,
		StockItemsCollection: *stockItems,
		TransferDate:         flatTransfer.TransferDate,
		Status:               flatTransfer.Status,
	}

	// var combinedItems []interface{}

	// for _, asset := range *assets {
	// 	combinedItems = append(combinedItems, asset)
	// }
	// for _, stock := range *stockItems {
	// 	combinedItems = append(combinedItems, stock)
	// }

	// transfer.ItemCollection = combinedItems

	return &transfer, nil
}

func (s *TransferService) GetTransfers() (*[]models.Transfer, error) {
	flatTransfers, err := s.tr.GetTransferRows()
	if err != nil {
		return nil, err
	}

	var transfers []models.Transfer

	for _, flatTransfer := range *flatTransfers {

		transfers = append(transfers, models.Transfer{
			ID: flatTransfer.ID,
			FromLocation: models.Location{
				ID:   flatTransfer.FromLocationID,
				Name: flatTransfer.FromLocationName,
			},
			ToLocation: models.Location{
				ID:   flatTransfer.ToLocationID,
				Name: flatTransfer.ToLocationName,
			},
			TransferDate: flatTransfer.TransferDate,
			Status:       flatTransfer.Status,
		})

	}
	return &transfers, nil
}
func (s *TransferService) RemoveStockItemFromTransfer(transferReq stocks.RemoveStockItemFromTransferRequest) error {
	return repository.WithTransaction(s.r.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		if err := decreaseStockInTransfer(tx, transferReq); err != nil {
			return err
		}

		if err := s.stockRepo.RemoveZeroQuantityStock(tx, transferReq); err != nil {
			return err
		}

		previousLocation, err := s.tr.GetTransferLocationById(tx, transferReq.TransferID)
		if err != nil {
			return err
		}

		if err := stocks.RestoreStockToLocation(tx, transferReq, previousLocation); err != nil {
			return err
		}

		return nil
	})
}

func (s *TransferService) handleSerializedItems(tx *goqu.TxDatabase, transferID int, assets []models.AssetItemRequest, locationID int, transitStatus string) error {
	if len(assets) == 0 {
		return nil
	}
	idList := mapToIDArray(assets)

	if err := s.tr.InsertSerializedItemTransferRecord(tx, transferID, idList); err != nil {
		return fmt.Errorf("failed to insert serialized asset transfer record: %w", err)
	}

	if err := s.tr.MoveSerializedItems(tx, idList, locationID, transitStatus); err != nil {
		return fmt.Errorf("failed to move serialized assets: %w", err)
	}

	return nil
}

func (s *TransferService) handleNonSerializedItems(tx *goqu.TxDatabase, transferID int, stocks []models.StockItemRequest, locationID, fromLocationID int) error {
	if len(stocks) == 0 {
		return nil
	}

	if err := s.tr.InsertNonSerializedItemTransferRecord(tx, transferID, stocks); err != nil {
		return fmt.Errorf("failed to insert non-serialized asset transfer record: %w", err)
	}

	if err := s.stockRepo.MoveNonSerializedItems(tx, stocks, locationID, fromLocationID); err != nil {
		return fmt.Errorf("failed to move non-serialized assets: %w", err)
	}

	return nil
}

type ValidationError struct {
	Message  string `json:"message"`
	Property string `json:"property"`
}

func (s *TransferService) ValidateStock(transferRequest models.TransferRequest) ([]ValidationError, error) {
	var validationState []ValidationError

	if len(transferRequest.AssetItemCollection) > 0 {
		assetIDs := mapToIDArray(transferRequest.AssetItemCollection)
		hasItemsOnStock, err := s.ar.HasItemsInLocation(assetIDs, transferRequest.FromLocationID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate serialized assets: %w", err)
		}
		if !hasItemsOnStock {
			validationState = append(validationState, ValidationError{
				Message:  "Serialized assets are not present in location",
				Property: "assets",
			})
		}
	}

	if len(transferRequest.StockItemCollection) > 0 {
		hasEnoughQuantity, err := s.tr.CanTransferNonSerializedItems(transferRequest.StockItemCollection, transferRequest.FromLocationID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate Stocks assets: %w", err)
		}

		if len(hasEnoughQuantity) != len(transferRequest.StockItemCollection) {
			validationState = append(validationState, ValidationError{
				Message:  "Non-serialized stocks are not present in location",
				Property: "stocks",
			})
		}
	}

	return validationState, nil
}

// Move to repo
func mapToIDArray(assetsReq []models.AssetItemRequest) []int {
	ids := make([]int, len(assetsReq))
	for i, item := range assetsReq {
		ids[i] = item.ID
	}
	return ids
}
