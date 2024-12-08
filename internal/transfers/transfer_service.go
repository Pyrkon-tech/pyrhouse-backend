package transfers

import (
	"fmt"
	"warehouse/internal/repository"
	"warehouse/internal/stocks"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
)

type TransferService struct {
	r         *repository.Repository
	tr        TransferRepository
	stockRepo *stocks.StockRepository
}

func (s *TransferService) PerformTransfer(req models.TransferRequest, transitStatus string) (int, error) {
	var transferID int

	err := repository.WithTransaction(s.r.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		var err error
		if transferID, err = s.tr.InsertTransferRecord(tx, req); err != nil {
			return fmt.Errorf("failed to insert transfer record: %w", err)
		}

		if err = s.handleSerializedItems(tx, transferID, req.SerialziedItemCollection, req.LocationID, transitStatus); err != nil {
			return err
		}

		if err = s.handleNonSerializedItems(tx, transferID, req.UnserializedItemCollection, req.LocationID, req.FromLocationID); err != nil {
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

	assets, err := s.r.GetTransferAssets(transferID)
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
		TransferDate:         flatTransfer.TransferDate,
		Status:               flatTransfer.Status,
		AssetsCollection:     *assets,
		StockItemsCollection: *stockItems,
	}

	return &transfer, nil
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

func (s *TransferService) handleSerializedItems(tx *goqu.TxDatabase, transferID int, assets []int, locationID int, transitStatus string) error {
	if len(assets) == 0 {
		return nil
	}

	if err := s.tr.InsertSerializedItemTransferRecord(tx, transferID, assets); err != nil {
		return fmt.Errorf("failed to insert serialized asset transfer record: %w", err)
	}

	if err := s.tr.MoveSerializedItems(tx, assets, locationID, transitStatus); err != nil {
		return fmt.Errorf("failed to move serialized assets: %w", err)
	}

	return nil
}

func (s *TransferService) handleNonSerializedItems(tx *goqu.TxDatabase, transferID int, assets []models.UnserializedItemRequest, locationID, fromLocationID int) error {
	if len(assets) == 0 {
		return nil
	}

	if err := s.tr.InsertNonSerializedItemTransferRecord(tx, transferID, assets); err != nil {
		return fmt.Errorf("failed to insert non-serialized asset transfer record: %w", err)
	}

	if err := s.stockRepo.MoveNonSerializedItems(tx, assets, locationID, fromLocationID); err != nil {
		return fmt.Errorf("failed to move non-serialized assets: %w", err)
	}

	return nil
}
