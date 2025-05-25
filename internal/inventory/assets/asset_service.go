package assets

import (
	"fmt"
	"log"
	"warehouse/internal/repository"
	"warehouse/pkg/auditlog"
	custom_error "warehouse/pkg/errors"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
)

type AssetService struct {
	assetsRepo *AssetsRepository
	repo       *repository.Repository
	auditLog   *auditlog.Auditlog
}

func NewAssetService(assetsRepo *AssetsRepository, repo *repository.Repository, auditLog *auditlog.Auditlog) *AssetService {
	return &AssetService{
		assetsRepo: assetsRepo,
		repo:       repo,
		auditLog:   auditLog,
	}
}

func (s *AssetService) CreateAssetsWithoutSerial(req models.EmergencyAssetRequest) ([]models.Asset, []string, error) {
	var createdAssets []models.Asset

	err := repository.WithTransaction(s.repo.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		for i := 0; i < req.Quantity; i++ {
			itemReq := models.ItemRequest{
				Serial:     nil,
				LocationId: req.LocationId,
				Status:     req.Status,
				CategoryId: req.CategoryId,
				Origin:     req.Origin,
			}

			asset, err := s.assetsRepo.PersistItem(itemReq)
			if err != nil {
				return fmt.Errorf("nie udało się utworzyć zasobu: %v", err)
			}

			pyrCode, err := s.assetsRepo.GenerateUniquePyrCode(asset.Category.ID, asset.Category.PyrID)
			if err != nil {
				if _, removeErr := s.assetsRepo.RemoveAsset(asset.ID); removeErr != nil {
					log.Printf("nie udało się usunąć zasobu po błędzie generowania kodu PYR: %v", removeErr)
				}
				return fmt.Errorf("nie udało się wygenerować kodu PYR: %v", err)
			}

			if err := s.assetsRepo.UpdatePyrCode(asset.ID, pyrCode); err != nil {
				if _, removeErr := s.assetsRepo.RemoveAsset(asset.ID); removeErr != nil {
					log.Printf("nie udało się usunąć zasobu po błędzie aktualizacji kodu PYR: %v", removeErr)
				}
				return fmt.Errorf("nie udało się zaktualizować kodu PYR: %v", err)
			}

			asset.PyrCode = pyrCode
			createdAssets = append(createdAssets, *asset)

			go s.auditLog.Log(
				"create",
				map[string]interface{}{
					"pyr_code":    asset.PyrCode,
					"location_id": asset.Location.ID,
					"msg":         "Utworzono zasób awaryjny bez numeru seryjnego",
				},
				asset,
			)
		}
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return createdAssets, nil, nil
}

func (s *AssetService) CreateBulkAssets(req models.BulkItemRequest) ([]models.Asset, []string, error) {
	var createdAssets []models.Asset
	var errors []string

	for _, serial := range req.Serials {
		itemReq := models.ItemRequest{
			Serial:     serial,
			LocationId: req.LocationId,
			Status:     req.Status,
			CategoryId: req.CategoryId,
			Origin:     req.Origin,
		}

		asset, err := s.assetsRepo.PersistItem(itemReq)
		if err != nil {
			switch err.(type) {
			case *custom_error.UniqueViolationError:
				errors = append(errors, fmt.Sprintf("Numer seryjny %s jest już zarejestrowany", *serial))
				continue
			default:
				errors = append(errors, fmt.Sprintf("Nie udało się utworzyć zasobu z numerem seryjnym %s: %v", *serial, err))
				continue
			}
		}

		pyrCode, err := s.assetsRepo.GenerateUniquePyrCode(asset.Category.ID, asset.Category.PyrID)
		if err != nil {
			log.Printf("Nie udało się wygenerować kodu PYR dla zasobu %d: %v", asset.ID, err)
			if _, removeErr := s.assetsRepo.RemoveAsset(asset.ID); removeErr != nil {
				log.Printf("Nie udało się usunąć zasobu po błędzie generowania kodu PYR: %v", removeErr)
			}
			errors = append(errors, fmt.Sprintf("Nie udało się wygenerować kodu PYR dla zasobu z numerem seryjnym %s: %v", *serial, err))
			continue
		}

		if err := s.assetsRepo.UpdatePyrCode(asset.ID, pyrCode); err != nil {
			log.Printf("Nie udało się zaktualizować kodu PYR dla zasobu %d: %v", asset.ID, err)
			if _, removeErr := s.assetsRepo.RemoveAsset(asset.ID); removeErr != nil {
				log.Printf("Nie udało się usunąć zasobu po błędzie aktualizacji kodu PYR: %v", removeErr)
			}
			errors = append(errors, fmt.Sprintf("Nie udało się zaktualizować kodu PYR dla zasobu z numerem seryjnym %s: %v", *serial, err))
			continue
		}

		asset.PyrCode = pyrCode
		createdAssets = append(createdAssets, *asset)

		go s.auditLog.Log(
			"create",
			map[string]interface{}{
				"serial":      asset.Serial,
				"pyr_code":    asset.PyrCode,
				"location_id": asset.Location.ID,
				"msg":         "Utworzono zasób zbiorczo",
			},
			asset,
		)
	}

	if len(errors) > 0 {
		err := s.RemoveAssets(createdAssets)
		return nil, errors, err
	}

	return createdAssets, errors, nil
}

func (s *AssetService) RemoveAssets(assets []models.Asset) error {
	for _, asset := range assets {
		if _, err := s.assetsRepo.RemoveAsset(asset.ID); err != nil {
			return fmt.Errorf("nie udało się usunąć zasobu %d: %v", asset.ID, err)
		}
	}
	return nil
}

func (s *AssetService) UpdateAssetLocation(assetID int, req models.DeliveryLocation) error {
	asset, err := s.assetsRepo.GetAsset(assetID)
	if err != nil {
		return fmt.Errorf("nie udało się pobrać zasobu: %v", err)
	}

	s.auditLog.Log(
		"last_known_location",
		map[string]interface{}{
			"asset_id": asset.ID,
			"msg":      "Ostatnia zarejestrowana lokalizacja",
			"location": map[string]interface{}{
				"location_id": asset.Location.ID,
				"latitude":    req.Lat,
				"longitude":   req.Lng,
				"timestamp":   req.Timestamp,
			},
		},
		asset,
	)

	return nil
}
