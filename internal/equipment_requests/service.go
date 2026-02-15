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
	questRepo        QuestRepositoryInterface // NEW: for Phase 2 DB persistence
	sheetID          string
	sheetName        string
	fuzzyThreshold   int
	categories       []models.ItemCategory // cached
}

func NewService(sheetReader *googlesheets.DutyScheduleService, categoryRepo *category.CategoryRepository, questRepo *Repository, sheetID, sheetName string, fuzzyThreshold int) *Service {
	return &Service{
		sheetReader:    sheetReader,
		categoryRepo:   categoryRepo,
		questRepo:      questRepo,
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

// ============================================================================
// Phase 2: Database Persistence Methods
// ============================================================================

// SyncStats tracks sync operation statistics
type SyncStats struct {
	Created      int
	Updated      int
	Unchanged    int
	ItemsAdded   int
	ItemsRemoved int
}

// SyncResult contains quests and sync statistics
type SyncResult struct {
	Quests []Quest
	Stats  *SyncStats
}

// SyncQuestsToDatabase fetches from sheet, aggregates, and persists to database
func (s *Service) SyncQuestsToDatabase(ctx context.Context) (*SyncResult, error) {
	startTime := time.Now()

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
		return &SyncResult{Quests: []Quest{}, Stats: &SyncStats{}}, nil
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

	// 4. Match categories with fuzzy matching
	for i := range sheetRows {
		match := s.matchCategoryWithFuzzy(ctx, sheetRows[i].Item)
		sheetRows[i].CategoryID = match.CategoryID
		sheetRows[i].CategoryMatch = match.MatchType
	}

	// 5. Aggregate into quests
	quests := s.aggregateQuests(sheetRows)

	// 6. Upsert quests to database
	stats := &SyncStats{}
	for i := range quests {
		// Set quest key for deduplication
		questKey := s.questKey(sheetRows[0]) // Use first row to get key fields
		quests[i].QuestKey = questKey

		if err := s.upsertQuest(ctx, &quests[i], stats); err != nil {
			log.Printf("[equipment-requests] Failed to upsert quest %s: %v", quests[i].ID, err)
			continue
		}
	}

	// 7. Log sync result
	duration := time.Since(startTime)
	syncLog := &SyncLog{
		RowsProcessed:   len(sheetRows),
		QuestsCreated:   stats.Created,
		QuestsUpdated:   stats.Updated,
		QuestsUnchanged: stats.Unchanged,
		ItemsAdded:      stats.ItemsAdded,
		Success:         true,
		DurationMs:      int(duration.Milliseconds()),
		SheetID:         s.sheetID,
	}

	if err := s.questRepo.CreateSyncLog(ctx, syncLog); err != nil {
		log.Printf("[equipment-requests] Failed to create sync log: %v", err)
	}

	log.Printf("[equipment-requests] Synced %d quests from %d rows (created: %d, updated: %d) in %dms",
		len(quests), len(sheetRows), stats.Created, stats.Updated, duration.Milliseconds())

	return &SyncResult{Quests: quests, Stats: stats}, nil
}

// upsertQuest creates or updates a quest in the database
func (s *Service) upsertQuest(ctx context.Context, quest *Quest, stats *SyncStats) error {
	// Try to get existing quest by key
	existing, err := s.questRepo.GetQuestByKey(ctx, quest.QuestKey)

	if err != nil || existing == nil {
		// Create new quest
		if err := s.questRepo.CreateQuest(ctx, quest); err != nil {
			return fmt.Errorf("failed to create quest: %w", err)
		}
		stats.Created++
		stats.ItemsAdded += len(quest.Items)
		log.Printf("[equipment-requests] Created new quest: %s", quest.ID)
	} else {
		// Update existing quest by quest_id
		if err := s.questRepo.UpdateQuest(ctx, existing.ID, quest); err != nil {
			return fmt.Errorf("failed to update quest: %w", err)
		}
		stats.Updated++
		stats.ItemsRemoved += len(existing.Items)
		stats.ItemsAdded += len(quest.Items)
		log.Printf("[equipment-requests] Updated quest: %s", quest.ID)
	}

	return nil
}

// matchCategoryWithFuzzy matches item name to category using fuzzy matching
func (s *Service) matchCategoryWithFuzzy(ctx context.Context, itemName string) CategoryMatch {
	// 1. Check manual mapping first (highest priority)
	if s.questRepo != nil {
		if categoryID, err := s.questRepo.GetCategoryMapping(ctx, itemName); err == nil && categoryID != nil {
			// Increment usage counter
			_ = s.questRepo.IncrementMappingUsage(ctx, itemName)
			return CategoryMatch{
				CategoryID: categoryID,
				MatchType:  "manual",
				Confidence: 1.0,
			}
		}
	}

	// 2. Exact match (case-sensitive)
	for _, cat := range s.categories {
		if cat.Name == itemName || cat.Label == itemName {
			return CategoryMatch{
				CategoryID: &cat.ID,
				MatchType:  "exact",
				Confidence: 1.0,
			}
		}
	}

	// 3. Fuzzy match using Levenshtein distance
	bestMatch := s.findBestFuzzyMatch(itemName)
	if bestMatch.Distance <= s.fuzzyThreshold && bestMatch.CategoryID != nil {
		confidence := 1.0 - (float64(bestMatch.Distance) / float64(len(itemName)))
		return CategoryMatch{
			CategoryID: bestMatch.CategoryID,
			MatchType:  "fuzzy",
			Confidence: confidence,
		}
	}

	// 4. No match
	return CategoryMatch{
		MatchType:  "none",
		Confidence: 0.0,
	}
}

// FuzzyMatchResult stores fuzzy match result
type FuzzyMatchResult struct {
	CategoryID *int
	Distance   int
}

// findBestFuzzyMatch finds the best category match using Levenshtein distance
func (s *Service) findBestFuzzyMatch(itemName string) FuzzyMatchResult {
	bestDistance := int(^uint(0) >> 1) // Max int
	var bestCategoryID *int

	for _, cat := range s.categories {
		// Try matching against both name and label
		nameDistance := levenshteinDistance(itemName, cat.Name)
		labelDistance := levenshteinDistance(itemName, cat.Label)

		minDistance := nameDistance
		if labelDistance < minDistance {
			minDistance = labelDistance
		}

		if minDistance < bestDistance {
			bestDistance = minDistance
			bestCategoryID = &cat.ID
		}
	}

	return FuzzyMatchResult{
		CategoryID: bestCategoryID,
		Distance:   bestDistance,
	}
}

// levenshteinDistance calculates the Levenshtein distance between two strings
// (minimum number of single-character edits required to change one string into the other)
func levenshteinDistance(s1, s2 string) int {
	// Convert to lowercase for case-insensitive matching
	s1 = toLower(s1)
	s2 = toLower(s2)

	len1 := len(s1)
	len2 := len(s2)

	// Create matrix
	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
	}

	// Initialize first row and column
	for i := 0; i <= len1; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len1][len2]
}

// Helper functions
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}
