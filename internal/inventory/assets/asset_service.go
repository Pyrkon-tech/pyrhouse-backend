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
	var errors []string

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
				errors = append(errors, fmt.Sprintf("Nie udało się utworzyć zasobu: %v", err))
				continue
			}

			pyrCode, err := s.assetsRepo.GenerateUniquePyrCode(asset.Category.ID, asset.Category.PyrID)
			if err != nil {
				log.Printf("Nie udało się wygenerować kodu PYR: %v", err)
				errors = append(errors, fmt.Sprintf("Nie udało się wygenerować kodu PYR: %v", err))
				if _, removeErr := s.assetsRepo.RemoveAsset(asset.ID); removeErr != nil {
					log.Printf("Nie udało się usunąć zasobu po błędzie generowania kodu PYR: %v", removeErr)
				}
				continue
			}

			if err := s.assetsRepo.UpdatePyrCode(asset.ID, pyrCode); err != nil {
				log.Printf("Nie udało się zaktualizować kodu PYR: %v", err)
				errors = append(errors, fmt.Sprintf("Nie udało się zaktualizować kodu PYR: %v", err))
				if _, removeErr := s.assetsRepo.RemoveAsset(asset.ID); removeErr != nil {
					log.Printf("Nie udało się usunąć zasobu po błędzie aktualizacji kodu PYR: %v", removeErr)
				}
				continue
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

	return createdAssets, errors, err
}

func (s *AssetService) CreateBulkAssets(req models.BulkItemRequest) ([]models.Asset, []string, error) {
	var createdAssets []models.Asset
	var errors []string

	err := repository.WithTransaction(s.repo.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
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
		return nil
	})

	return createdAssets, errors, err
}
