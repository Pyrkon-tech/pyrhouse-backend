package stocks

import (
	"warehouse/internal/auditlog"
	"warehouse/internal/models"
	"warehouse/internal/repository"
)

// StockService reprezentuje serwis dla operacji na elementach magazynowych
type StockService struct {
	repository    *repository.Repository
	stockRepo     *StockRepository
	auditLog      *auditlog.Auditlog
	OnStockChanged func(locationID int, action string) // optional — wired by DI container
}

// NewStockService tworzy nową instancję StockService
func NewStockService(repo *repository.Repository, stockRepo *StockRepository, auditLog *auditlog.Auditlog) *StockService {
	return &StockService{
		repository: repo,
		stockRepo:  stockRepo,
		auditLog:   auditLog,
	}
}

// CreateStockItem tworzy nowy element magazynowy
func (s *StockService) CreateStockItem(req models.CreateStockItemRequest) (*models.StockItem, error) {
	createdStock, err := s.stockRepo.PersistStockItem(req)
	if err != nil {
		return nil, err
	}

	// Logowanie operacji
	go s.auditLog.Log(
		"create",
		map[string]interface{}{
			"category_id": createdStock.Category.ID,
			"quantity":    createdStock.Quantity,
			"origin":      createdStock.Origin,
			"msg":         "Stock item created successfully",
		},
		createdStock,
	)

	if s.OnStockChanged != nil {
		go s.OnStockChanged(req.LocationID, "created")
	}

	return createdStock, nil
}

// UpdateStockItem aktualizuje element magazynowy
func (s *StockService) UpdateStockItem(req *models.PatchStockItemRequest) (*models.StockItem, error) {
	updatedStock, err := s.stockRepo.UpdateStock(req)
	if err != nil {
		return nil, err
	}

	// Logowanie operacji
	go s.auditLog.Log(
		"update",
		map[string]interface{}{
			"stock_id": req.ID,
			"msg":      "Stock item updated",
		},
		updatedStock,
	)

	if s.OnStockChanged != nil {
		go s.OnStockChanged(updatedStock.Location.ID, "updated")
	}

	return updatedStock, nil
}

// GetStockItems pobiera listę elementów magazynowych
func (s *StockService) GetStockItems() (*[]models.StockItem, error) {
	return s.stockRepo.GetStockItems()
}

// GetStockItemByID pobiera element magazynowy po ID
func (s *StockService) GetStockItemByID(id int) (*models.StockItem, error) {
	return s.stockRepo.GetStockItem(id)
}

// GetStockItemsBy pobiera elementy magazynowe spełniające warunki
func (s *StockService) GetStockItemsBy(conditions repository.QueryBuilder) (*[]models.StockItem, error) {
	return s.stockRepo.GetStockItemsBy(conditions)
}

// DeleteStock usuwa element magazynowy
func (s *StockService) DeleteStock(id int) error {
	stock, _ := s.stockRepo.GetStockItem(id)

	err := s.stockRepo.DeleteStock(id)
	if err != nil {
		return err
	}

	if stock != nil {
		go s.auditLog.Log(
			"delete",
			map[string]interface{}{
				"stock_id": id,
				"msg":      "Stock item deleted",
			},
			stock,
		)

		if s.OnStockChanged != nil {
			go s.OnStockChanged(stock.Location.ID, "deleted")
		}
	}

	return nil
}
