package releases

import (
	"fmt"
	"warehouse/internal/auditlog"
	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
)

type Service struct {
	repo     *Repository
	baseRepo *repository.Repository
	auditLog *auditlog.Auditlog
}

func NewService(repo *Repository, baseRepo *repository.Repository, auditLog *auditlog.Auditlog) *Service {
	return &Service{
		repo:     repo,
		baseRepo: baseRepo,
		auditLog: auditLog,
	}
}

// Suggest returns assets and stocks for a given origin (and optional location).
func (s *Service) Suggest(originID int, locationID *int) (*SuggestResponse, error) {
	assets, err := s.repo.SuggestAssets(originID, locationID)
	if err != nil {
		return nil, err
	}
	stocks, err := s.repo.SuggestStocks(originID, locationID)
	if err != nil {
		return nil, err
	}
	return &SuggestResponse{Assets: assets, Stocks: stocks}, nil
}

// CreateRelease creates a draft release with the given items.
func (s *Service) CreateRelease(req CreateReleaseRequest, userID int) (*ReleaseDetail, error) {
	// Validate assets
	if err := s.repo.ValidateAssetsForRelease(req.Assets, nil); err != nil {
		return nil, fmt.Errorf("asset validation failed: %w", err)
	}
	// Validate stocks
	if err := s.repo.ValidateStocksForRelease(req.Stocks); err != nil {
		return nil, fmt.Errorf("stock validation failed: %w", err)
	}

	var releaseID int
	err := repository.WithTransaction(s.baseRepo.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		ref, err := s.repo.GenerateReference()
		if err != nil {
			return err
		}

		release, err := s.repo.CreateRelease(tx, ref, req.ReleasedTo, req.OriginID, req.Notes, userID)
		if err != nil {
			return err
		}
		releaseID = release.ID

		if err := s.repo.InsertReleaseAssets(tx, release.ID, req.Assets); err != nil {
			return err
		}
		if err := s.repo.InsertReleaseStocks(tx, release.ID, req.Stocks); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.GetReleaseDetail(releaseID)
}

// UpdateItems replaces the items in a draft release.
func (s *Service) UpdateItems(releaseID int, req UpdateItemsRequest) (*ReleaseDetail, error) {
	release, err := s.repo.GetRelease(releaseID)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return nil, fmt.Errorf("release not found")
	}
	if release.Status != "draft" {
		return nil, fmt.Errorf("can only update items in draft releases")
	}

	if err := s.repo.ValidateAssetsForRelease(req.Assets, &releaseID); err != nil {
		return nil, fmt.Errorf("asset validation failed: %w", err)
	}
	if err := s.repo.ValidateStocksForRelease(req.Stocks); err != nil {
		return nil, fmt.Errorf("stock validation failed: %w", err)
	}

	err = repository.WithTransaction(s.baseRepo.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		if err := s.repo.ClearReleaseItems(tx, releaseID); err != nil {
			return err
		}
		if err := s.repo.InsertReleaseAssets(tx, releaseID, req.Assets); err != nil {
			return err
		}
		if err := s.repo.InsertReleaseStocks(tx, releaseID, req.Stocks); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.GetReleaseDetail(releaseID)
}

// Confirm executes the release: snapshots data, deletes assets, decreases stocks.
func (s *Service) Confirm(releaseID int) (*ReleaseDetail, error) {
	release, err := s.repo.GetRelease(releaseID)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return nil, fmt.Errorf("release not found")
	}
	if release.Status != "draft" {
		return nil, fmt.Errorf("can only confirm draft releases")
	}

	// Re-validate before confirming
	assets, err := s.repo.GetReleaseAssets(releaseID)
	if err != nil {
		return nil, err
	}
	assetIDs := make([]int, len(assets))
	for i, a := range assets {
		assetIDs[i] = a.ItemID
	}
	if err := s.repo.ValidateAssetsForRelease(assetIDs, &releaseID); err != nil {
		return nil, fmt.Errorf("asset validation failed on confirm: %w", err)
	}

	stocks, err := s.repo.GetReleaseStocks(releaseID)
	if err != nil {
		return nil, err
	}
	stockReqs := make([]StockReleaseReq, len(stocks))
	for i, st := range stocks {
		stockReqs[i] = StockReleaseReq{StockID: st.StockID, Quantity: st.Quantity}
	}
	if err := s.repo.ValidateStocksForRelease(stockReqs); err != nil {
		return nil, fmt.Errorf("stock validation failed on confirm: %w", err)
	}

	err = repository.WithTransaction(s.baseRepo.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		return s.repo.ConfirmRelease(tx, releaseID)
	})
	if err != nil {
		return nil, err
	}

	return s.GetReleaseDetail(releaseID)
}

// GetReleaseDetail returns a full release with assets, stocks, and summary.
func (s *Service) GetReleaseDetail(releaseID int) (*ReleaseDetail, error) {
	release, err := s.repo.GetRelease(releaseID)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return nil, nil
	}

	assets, err := s.repo.GetReleaseAssets(releaseID)
	if err != nil {
		return nil, err
	}
	stocks, err := s.repo.GetReleaseStocks(releaseID)
	if err != nil {
		return nil, err
	}

	totalStockQty := 0
	for _, st := range stocks {
		totalStockQty += st.Quantity
	}

	return &ReleaseDetail{
		Release: *release,
		Assets:  assets,
		Stocks:  stocks,
		Summary: ReleaseSummary{
			TotalAssets:        len(assets),
			TotalStockQuantity: totalStockQty,
		},
	}, nil
}

// ListReleases returns releases with optional filters.
func (s *Service) ListReleases(status *string, originID *int) ([]Release, error) {
	return s.repo.ListReleases(status, originID)
}

// DeleteRelease removes a draft release.
func (s *Service) DeleteRelease(releaseID int) error {
	return s.repo.DeleteRelease(releaseID)
}
