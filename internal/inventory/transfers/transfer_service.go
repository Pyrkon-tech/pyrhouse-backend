package transfers

import (
	"fmt"
	"log"
	"warehouse/internal/inventory/assets"
	inventorylog "warehouse/internal/inventory/inventory_log"
	"warehouse/internal/inventory/stocks"
	"warehouse/internal/repository"
	"warehouse/pkg/metadata"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
)

type TransferService struct {
	r         *repository.Repository
	tr        TransferRepository
	ar        *assets.AssetsRepository
	stockRepo *stocks.StockRepository
	il        *inventorylog.InventoryLog
}

type ValidationError struct {
	Message  string `json:"message"`
	Property string `json:"property"`
}

func NewService(r *repository.Repository, tr TransferRepository, ar *assets.AssetsRepository, sr *stocks.StockRepository, il *inventorylog.InventoryLog) *TransferService {
	return &TransferService{
		r:         r,
		tr:        tr,
		ar:        ar,
		stockRepo: sr,
		il:        il,
	}
}

func (s *TransferService) InitTransfer(req models.TransferRequest, transitStatus string) (int, error) {
	var transferID int

	err := repository.WithTransaction(s.r.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		var err error
		if transferID, err = s.tr.InsertTransferRecord(tx, req); err != nil {
			return fmt.Errorf("failed to insert transfer record: %w", err)
		}

		if err = s.startAssetsTransfer(tx, transferID, req.AssetItemCollection, req.LocationID, transitStatus); err != nil {
			return err
		}

		if err = s.startStockItemsTransfer(tx, transferID, req.StockItemCollection, req.FromLocationID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	go s.createInventoryLog("in_transfer", transferID)

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
	var err error

	return repository.WithTransaction(s.r.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		if err = decreaseStockInTransfer(tx, transferReq); err != nil {
			return err
		}

		if err = s.stockRepo.RemoveZeroQuantityStock(tx, transferReq); err != nil {
			return err
		}

		if transferReq.ToLocationID == 0 {
			transferReq.ToLocationID, err = s.tr.GetTransferLocationById(tx, transferReq.TransferID)
			if err != nil {
				return err
			}
		}
		if err = s.stockRepo.RestoreStockToLocation(tx, transferReq); err != nil {
			return err
		}

		return nil
	})
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

func (s *TransferService) completeStockItemsTransfer(tx *goqu.TxDatabase, transferID int) error {

	HasStockItems, err := s.tr.HasStockItemsInTransfer(tx, transferID)
	if err != nil {
		return err
	}

	if !HasStockItems {
		return nil
	}

	if err := s.stockRepo.IncreaseStockAtDestination(tx, transferID); err != nil {
		return fmt.Errorf("failed to increase stock items at destination: %w", err)
	}
	// TODO Prevent remove or figure out a way to keep what was transfered
	if err := s.tr.RemoveStockItemsTransferRecords(tx, transferID); err != nil {
		return fmt.Errorf("failed remove stock items transfer entry: %w", err)
	}

	return nil
}

func (s *TransferService) startAssetsTransfer(tx *goqu.TxDatabase, transferID int, assets []models.AssetItemRequest, locationID int, transitStatus string) error {
	if len(assets) == 0 {
		return nil
	}
	idList := mapToIDArray(assets)

	if err := s.tr.InsertAssetsTransferRecord(tx, transferID, idList); err != nil {
		return fmt.Errorf("failed to insert serialized asset transfer record: %w", err)
	}

	if err := s.tr.MoveAssets(tx, idList, locationID, transitStatus); err != nil {
		return fmt.Errorf("failed to move serialized assets: %w", err)
	}

	return nil
}

func (s *TransferService) startStockItemsTransfer(tx *goqu.TxDatabase, transferID int, stocks []models.StockItemRequest, fromLocationID int) error {
	if len(stocks) == 0 {
		return nil
	}

	// TODO Prevent remove or figure out a way to keep what was transfered
	if err := s.tr.InsertStockItemsTransferRecord(tx, transferID, stocks); err != nil {
		return fmt.Errorf("failed to insert non-serialized asset transfer record: %w", err)
	}
	if err := s.stockRepo.DecreaseStockItemsQuantity(tx, stocks, fromLocationID); err != nil {
		return fmt.Errorf("failed to move non-serialized assets: %w", err)
	}

	return nil
}

func (s *TransferService) ConfirmTransfer(transferID int, status string) error {
	var err error
	// TODO get only ids?
	assets, err := s.ar.GetTransferAssets(transferID)
	assetIDs := func(assets []models.Asset) []int {
		var ids []int
		for _, asset := range assets {
			ids = append(ids, asset.ID)
		}
		return ids
	}(*assets)

	err = repository.WithTransaction(s.r.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		if len(assetIDs) > 0 {
			if err := s.ar.UpdateItemStatus(assetIDs, metadata.StatusLocated, tx); err != nil {
				return fmt.Errorf("unable to update assets err: %w", err)
			}
		}

		if err := s.completeStockItemsTransfer(tx, transferID); err != nil {
			return fmt.Errorf("unable to update stock items err: %w", err)
		}

		err = s.tr.UpdateTransferStatus(transferID, status)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	go s.createInventoryLog("delivered", transferID)

	return nil
}

func (s *TransferService) createInventoryLog(action string, transferID int) {

	transfer, err := s.GetTransfer(transferID)

	if err != nil {
		log.Printf("Unable to get transfer id: %d for auditlog error: %s", transferID, err.Error())
	}

	s.il.CreateTransferAuditLogEntry(action, transfer)
}

// TODO decide if move to repo
func mapToIDArray(assetsReq []models.AssetItemRequest) []int {
	ids := make([]int, len(assetsReq))
	for i, item := range assetsReq {
		ids[i] = item.ID
	}
	return ids
}

func (s *TransferService) CancelTransfer(transfer *models.Transfer) error {
	err := repository.WithTransaction(s.r.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		// Pobierz aktywa w jednym zapytaniu
		assets, err := s.ar.GetTransferAssets(transfer.ID)
		if err != nil {
			return fmt.Errorf("failed to get transfer assets: %w", err)
		}

		// Aktualizuj status aktywów wsadowo
		if len(*assets) > 0 {
			assetIDs := make([]int, len(*assets))
			for i, asset := range *assets {
				assetIDs[i] = asset.ID
			}

			// Przywróć aktywa do oryginalnej lokalizacji i zaktualizuj status
			for _, assetID := range assetIDs {
				if err := s.ar.UpdateAssetStatusAndLocation(tx, assetID, transfer.FromLocation.ID, metadata.StatusLocated); err != nil {
					return fmt.Errorf("failed to restore asset %d to original location: %w", assetID, err)
				}
			}

			// Dodaj wpisy do logu asynchronicznie
			go func(assets []models.Asset) {
				for _, asset := range assets {
					s.il.CreateAssetAuditLogEntry("cancelled", &asset, "Asset returned to original location")
				}
			}(*assets)
		}

		// Sprawdź i przywróć pozycje magazynowe
		hasStockItems, err := s.tr.HasStockItemsInTransfer(tx, transfer.ID)
		if err != nil {
			return fmt.Errorf("failed to check stock items in transfer: %w", err)
		}

		if hasStockItems {
			stockItems, err := s.stockRepo.GetStockItemsByTransfer(transfer.ID)
			if err != nil {
				return fmt.Errorf("failed to get stock items: %w", err)
			}

			// Przywróć pozycje magazynowe wsadowo
			for _, item := range *stockItems {
				if err := s.stockRepo.RestoreStockToLocation(tx, stocks.RemoveStockItemFromTransferRequest{
					CategoryID:   item.Category.ID,
					TransferID:   transfer.ID,
					Quantity:     item.Quantity,
					ToLocationID: transfer.FromLocation.ID,
				}); err != nil {
					return fmt.Errorf("failed to restore stock item %d to original location: %w", item.Category.ID, err)
				}
			}
		}

		// Zaktualizuj status transferu
		if err := s.tr.UpdateTransferStatus(transfer.ID, "cancelled"); err != nil {
			return fmt.Errorf("failed to update transfer status: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Dodaj wpis do logu inwentaryzacji poza transakcją
	go s.createInventoryLog("cancelled", transfer.ID)

	return nil
}
