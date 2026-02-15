package equipment_requests

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"time"

	"warehouse/internal/integrations/googlesheets"
	"warehouse/internal/inventory/category"
	"warehouse/internal/models"
)

type Service struct {
	sheetReader      *googlesheets.DutyScheduleService
	categoryRepo     *category.CategoryRepository
	sheetID          string
	sheetName        string
	fuzzyThreshold   int
	categories       []models.ItemCategory // cached
}

func NewService(sheetReader *googlesheets.DutyScheduleService, categoryRepo *category.CategoryRepository, sheetID, sheetName string, fuzzyThreshold int) *Service {
	return &Service{
		sheetReader:    sheetReader,
		categoryRepo:   categoryRepo,
		sheetID:        sheetID,
		sheetName:      sheetName,
		fuzzyThreshold: fuzzyThreshold,
	}
}

// SyncQuests fetches sheet data and aggregates into quests
func (s *Service) SyncQuests(ctx context.Context) ([]Quest, error) {
	// 1. Fetch categories (cache for matching)
	if err := s.loadCategories(); err != nil {
		return nil, fmt.Errorf("failed to load categories: %w", err)
	}

	// 2. Fetch sheet data
	rows, err := s.sheetReader.FetchSheet(s.sheetID, s.sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sheet: %w", err)
	}

	if len(rows) < 2 {
		return []Quest{}, nil // Empty or header-only sheet
	}

	// 3. Parse rows
	mapper := NewColumnMapper(rows[0])
	if !mapper.HasRequiredColumns() {
		return nil, fmt.Errorf("missing required columns: %v", mapper.MissingColumns())
	}

	sheetRows := []SheetRow{}
	for i := 1; i < len(rows); i++ {
		sr, err := mapper.ParseRow(rows[i], i+1)
		if err != nil {
			log.Printf("[equipment-requests] Skipping row %d: %v", i+1, err)
			continue
		}
		sheetRows = append(sheetRows, *sr)
	}

	// 4. Match categories
	for i := range sheetRows {
		match := s.matchCategory(sheetRows[i].Item)
		sheetRows[i].CategoryID = match.CategoryID
		sheetRows[i].CategoryMatch = match.MatchType
	}

	// 5. Aggregate into quests
	quests := s.aggregateQuests(sheetRows)

	log.Printf("[equipment-requests] Synced %d quests from %d rows", len(quests), len(sheetRows))
	return quests, nil
}

func (s *Service) loadCategories() error {
	cats, err := s.categoryRepo.GetCategories()
	if err != nil {
		return err
	}
	s.categories = cats
	return nil
}

func (s *Service) matchCategory(itemName string) CategoryMatch {
	// TODO: Implement fuzzy matching (Levenshtein)
	// For now: exact match only
	for _, cat := range s.categories {
		if cat.Name == itemName || cat.Label == itemName {
			return CategoryMatch{
				CategoryID: &cat.ID,
				MatchType:  "exact",
				Confidence: 1.0,
			}
		}
	}

	return CategoryMatch{
		MatchType:  "none",
		Confidence: 0.0,
	}
}

func (s *Service) aggregateQuests(rows []SheetRow) []Quest {
	questMap := make(map[string]*Quest)

	for _, row := range rows {
		// Skip non-ordered items
		if row.Status != StatusOrdered {
			continue
		}

		// Generate quest key
		key := s.questKey(row)

		// Get or create quest
		quest, exists := questMap[key]
		if !exists {
			quest = &Quest{
				ID: generateQuestID(key),
				Destination: Destination{
					Pavilion: row.Pavilion,
					Location: row.Location,
				},
				Recipient:    row.Recipient,
				DeliveryDate: row.DeliveryDate,
				PickupTime:   row.PickupTime,
				BudgetOwner:  row.BudgetOwner,
				Status:       "pending",
				Items:        []QuestItem{},
				SourceRows:   []int{},
				LastSynced:   time.Now(),
			}
			questMap[key] = quest
		}

		// Add item to quest
		quest.Items = append(quest.Items, QuestItem{
			Name:          row.Item,
			Quantity:      row.Quantity,
			CategoryID:    row.CategoryID,
			CategoryMatch: row.CategoryMatch,
			Notes:         row.Notes,
		})
		quest.SourceRows = append(quest.SourceRows, row.RowNumber)
	}

	// Convert map to slice
	quests := make([]Quest, 0, len(questMap))
	for _, q := range questMap {
		quests = append(quests, *q)
	}

	return quests
}

func (s *Service) questKey(row SheetRow) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s",
		row.Pavilion,
		row.Location,
		row.Recipient,
		row.DeliveryDate,
		row.PickupTime,
	)
}

func generateQuestID(key string) string {
	hash := md5.Sum([]byte(key))
	return fmt.Sprintf("quest-%x", hash[:8])
}
