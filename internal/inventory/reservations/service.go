package reservations

import (
	"context"
	"fmt"
	"warehouse/internal/auditlog"
	"warehouse/internal/inventory/assets"
	"warehouse/internal/models"
	"warehouse/internal/origins"
	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
)

type Service struct {
	repo      *Repository
	assetRepo *assets.AssetsRepository
	baseRepo  *repository.Repository
	originSvc *origins.Service
	auditLog  *auditlog.Auditlog
}

func NewService(
	repo *Repository,
	assetRepo *assets.AssetsRepository,
	baseRepo *repository.Repository,
	originSvc *origins.Service,
	auditLog *auditlog.Auditlog,
) *Service {
	return &Service{
		repo:      repo,
		assetRepo: assetRepo,
		baseRepo:  baseRepo,
		originSvc: originSvc,
		auditLog:  auditLog,
	}
}

func (s *Service) Reserve(ctx context.Context, req ReserveRequest) ([]PyrCodeReservation, error) {
	categoryType, err := s.baseRepo.GetCategoryType(req.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("invalid category: %w", err)
	}
	if categoryType != "asset" {
		return nil, fmt.Errorf("category must be of type asset, got %q", categoryType)
	}

	var pyrID string
	_, err = s.baseRepo.GoquDBWrapper.
		Select(goqu.L("COALESCE(pyr_id, '')")).
		From("item_category").
		Where(goqu.Ex{"id": req.CategoryID}).
		Executor().ScanVal(&pyrID)
	if err != nil {
		return nil, fmt.Errorf("failed to get category pyr_id: %w", err)
	}
	if pyrID == "" {
		return nil, fmt.Errorf("category %d has no pyr_id configured", req.CategoryID)
	}

	var created []PyrCodeReservation
	for i := 0; i < req.Quantity; i++ {
		code, err := s.assetRepo.GenerateUniquePyrCode(req.CategoryID, pyrID)
		if err != nil {
			return nil, fmt.Errorf("failed to generate pyr_code #%d: %w", i+1, err)
		}
		batch, err := s.repo.CreateReservations([]string{code}, req.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("failed to persist reservations: %w", err)
		}
		created = append(created, batch[0])
	}

	return created, nil
}

func (s *Service) GetReservations(ctx context.Context, categoryID *int, status string) ([]PyrCodeReservation, error) {
	return s.repo.GetReservations(categoryID, status)
}

func (s *Service) Delete(ctx context.Context, pyrCodes []string, ids []int) (int, error) {
	if len(pyrCodes) == 0 && len(ids) == 0 {
		return 0, fmt.Errorf("provide pyr_codes or ids to delete")
	}

	var deleted int
	if len(pyrCodes) > 0 {
		n, claimed, err := s.repo.DeleteByPyrCodes(pyrCodes)
		if err != nil {
			return 0, err
		}
		if len(claimed) > 0 {
			return 0, &ClaimedError{PyrCodes: claimed}
		}
		deleted += n
	}
	if len(ids) > 0 {
		n, claimed, err := s.repo.DeleteByIDs(ids)
		if err != nil {
			return 0, err
		}
		if len(claimed) > 0 {
			return 0, &ClaimedError{PyrCodes: claimed}
		}
		deleted += n
	}
	return deleted, nil
}

func (s *Service) Claim(ctx context.Context, req ClaimRequest) ([]models.Asset, error) {
	if req.LocationID == 0 {
		req.LocationID = 1
	}

	resolution, err := s.originSvc.ResolveOrigin(ctx, req.Origin)
	if err != nil {
		return nil, fmt.Errorf("invalid origin: %w", err)
	}

	pyrCodes := make([]string, len(req.Items))
	for i, item := range req.Items {
		pyrCodes[i] = item.PyrCode
	}

	var createdAssets []models.Asset

	err = repository.WithTransaction(s.baseRepo.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		reservations, err := s.repo.FindUnclaimedByPyrCodes(tx, pyrCodes)
		if err != nil {
			return err
		}

		resMap := make(map[string]PyrCodeReservation, len(reservations))
		for _, r := range reservations {
			resMap[r.PyrCode] = r
		}

		var claimErrors []ClaimError
		for _, code := range pyrCodes {
			if _, ok := resMap[code]; !ok {
				claimErrors = append(claimErrors, ClaimError{PyrCode: code, Reason: "reservation not found or already claimed"})
			}
		}
		if len(claimErrors) > 0 {
			return &ValidationError{Errors: claimErrors}
		}

		itemMap := make(map[string]ClaimItem, len(req.Items))
		for _, item := range req.Items {
			itemMap[item.PyrCode] = item
		}

		for _, res := range reservations {
			claimItem := itemMap[res.PyrCode]

			asset, err := s.assetRepo.PersistItem(models.ItemRequest{
				Serial:       claimItem.Serial,
				LocationId:   req.LocationID,
				Status:       "available",
				CategoryId:   res.CategoryID,
				OriginID:     &resolution.OriginID,
				OriginSuffix: resolution.OriginSuffix,
			})
			if err != nil {
				return fmt.Errorf("failed to create asset for %s: %w", res.PyrCode, err)
			}

			if err := s.assetRepo.UpdatePyrCode(asset.ID, res.PyrCode); err != nil {
				return fmt.Errorf("failed to set pyr_code for asset %d: %w", asset.ID, err)
			}
			asset.PyrCode = res.PyrCode

			if err := s.repo.MarkClaimed(tx, res.ID, asset.ID); err != nil {
				return err
			}

			createdAssets = append(createdAssets, *asset)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	go func() {
		for i := range createdAssets {
			a := createdAssets[i]
			s.auditLog.Log("create", map[string]interface{}{
				"serial":   a.Serial,
				"pyr_code": a.PyrCode,
				"msg":      "Asset created via pyr_code reservation claim",
			}, &a)
		}
	}()

	return createdAssets, nil
}

// ClaimedError is returned when DELETE targets already-claimed reservations.
type ClaimedError struct {
	PyrCodes []string
}

func (e *ClaimedError) Error() string {
	return fmt.Sprintf("cannot delete claimed reservations: %v", e.PyrCodes)
}

// ValidationError is returned when claim validation fails.
type ValidationError struct {
	Errors []ClaimError
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("claim validation failed for %d pyr_code(s)", len(e.Errors))
}
