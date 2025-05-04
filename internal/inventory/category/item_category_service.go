package category

import (
	"fmt"
	"warehouse/internal/repository"
	"warehouse/pkg/models"
)

type ItemCategoryService struct {
	repository *repository.Repository
}

func NewItemCategoryService(r *repository.Repository) *ItemCategoryService {
	return &ItemCategoryService{
		repository: r,
	}
}

func (s *ItemCategoryService) CreateCategory(category models.ItemCategory) (*models.ItemCategory, error) {
	if category.Label == "" {
		return nil, fmt.Errorf("label jest wymagane")
	}

	if category.Type == "" {
		category.Type = "asset" // domyślny typ
	}

	category.GenerateNameFromLabel()

	if category.PyrID == "" {
		if err := s.generateUniquePyrID(&category); err != nil {
			return nil, err
		}
	} else {
		isUnique, err := s.repository.CheckPyrIDUniqueness(category.PyrID, nil)
		if err != nil {
			return nil, fmt.Errorf("nie udało się sprawdzić unikalności PyrID: %w", err)
		}
		if !isUnique {
			return nil, fmt.Errorf("PyrID '%s' jest już używane", category.PyrID)
		}
	}

	return s.repository.PersistItemCategory(category)
}

func (s *ItemCategoryService) UpdateCategory(categoryID int, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return fmt.Errorf("brak pól do aktualizacji")
	}

	if pyrID, ok := updates["pyr_id"].(string); ok {
		isUnique, err := s.repository.CheckPyrIDUniqueness(pyrID, &categoryID)
		if err != nil {
			return fmt.Errorf("nie udało się sprawdzić unikalności PyrID: %w", err)
		}
		if !isUnique {
			return fmt.Errorf("PyrID '%s' jest już używane", pyrID)
		}
	}

	return s.repository.UpdateItemCategory(categoryID, updates)
}

func (s *ItemCategoryService) GetCategories() (*[]models.ItemCategory, error) {
	return s.repository.GetCategories()
}

func (s *ItemCategoryService) DeleteCategory(categoryID string) error {
	return s.repository.DeleteItemCategoryByID(categoryID)
}

func (s *ItemCategoryService) GetCategoryType(categoryID int) (string, error) {
	return s.repository.GetCategoryType(categoryID)
}

func (s *ItemCategoryService) generateUniquePyrID(category *models.ItemCategory) error {
	if category.PyrID == "" {
		category.GeneratePyrID()
	}

	// Najpierw sprawdzamy czy 3-literowy kod jest wolny
	isUnique, err := s.repository.CheckPyrIDUniqueness(category.PyrID, nil)
	if err != nil {
		return fmt.Errorf("nie udało się sprawdzić unikalności PyrID: %w", err)
	}

	if !isUnique {
		// Jeśli kod jest zajęty, próbujemy dodać cyfry od 1 do 9
		for i := 1; i <= 9; i++ {
			newPyrID := fmt.Sprintf("%s%d", category.PyrID, i)
			isUnique, err = s.repository.CheckPyrIDUniqueness(newPyrID, nil)
			if err != nil {
				return fmt.Errorf("nie udało się sprawdzić unikalności PyrID: %w", err)
			}
			if isUnique {
				category.PyrID = newPyrID
				return nil
			}
		}
		// Jeśli wszystkie cyfry są zajęte, generujemy nowy 3-literowy kod
		category.GeneratePyrID()
		// Dodajemy cyfrę 1 do nowego kodu
		category.PyrID = fmt.Sprintf("%s1", category.PyrID)
	}

	return nil
}
