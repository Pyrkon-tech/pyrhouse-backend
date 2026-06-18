package equipment_requests

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"warehouse/internal/inventory/category"
	"warehouse/internal/models"
	"warehouse/internal/settings"
)

// TransferCreator abstracts transfer creation to avoid circular dependency with transfers package
type TransferCreator interface {
	InitTransfer(req models.TransferRequest, transitStatus string) (int, error)
}

// SheetReader abstracts Google Sheets access so tests can inject fake data.
type SheetReader interface {
	FetchSheet(spreadsheetID, sheetName string) ([][]string, error)
}

type Service struct {
	sheetReader     SheetReader
	categoryRepo    *category.CategoryRepository
	questRepo       QuestRepositoryInterface
	transferCreator TransferCreator
	settingsRepo    *settings.Repository
	fallbackSheetID   string
	fallbackSheetName string
	fuzzyThreshold  int
	categories      []models.ItemCategory // cached

	// SSE broadcaster
	sseMu      sync.RWMutex
	sseClients map[chan QuestEvent]struct{}

	// Dispatch SSE hooks — set via SetDispatchHooks in DI.
	// onTransferDispatched is called when a transfer is created from a quest (on_mission).
	// onTransferEnded is called when a transfer is completed or cancelled (available).
	onTransferDispatched func(transferID int)
	onTransferEnded      func(transferID int)
}

// SetDispatchHooks wires dispatch SSE broadcast callbacks (avoids circular DI).
func (s *Service) SetDispatchHooks(dispatched, ended func(transferID int)) {
	s.onTransferDispatched = dispatched
	s.onTransferEnded = ended
}

func NewService(
	sheetReader SheetReader,
	categoryRepo *category.CategoryRepository,
	questRepo *Repository,
	settingsRepo *settings.Repository,
	fallbackSheetID, fallbackSheetName string,
	fuzzyThreshold int,
) *Service {
	return &Service{
		sheetReader:       sheetReader,
		categoryRepo:      categoryRepo,
		questRepo:         questRepo,
		settingsRepo:      settingsRepo,
		fallbackSheetID:   fallbackSheetID,
		fallbackSheetName: fallbackSheetName,
		fuzzyThreshold:    fuzzyThreshold,
		sseClients:        make(map[chan QuestEvent]struct{}),
	}
}

// getSheetConfig reads sheet ID and name from DB, falling back to env values
func (s *Service) getSheetConfig(ctx context.Context) (sheetID, sheetName string, err error) {
	sheetID, _ = s.settingsRepo.Get(ctx, "equipment_request.sheet_id")
	sheetName, _ = s.settingsRepo.Get(ctx, "equipment_request.sheet_name")

	if sheetID == "" {
		sheetID = s.fallbackSheetID
	}
	if sheetName == "" {
		sheetName = s.fallbackSheetName
	}

	if sheetID == "" {
		return "", "", fmt.Errorf("equipment_request.sheet_id not configured")
	}
	return sheetID, sheetName, nil
}

// ============================================================================
// SSE Broadcaster
// ============================================================================

// Subscribe registers a channel to receive quest events over SSE.
// The returned channel is buffered (capacity 10) to avoid blocking the sync.
func (s *Service) Subscribe() chan QuestEvent {
	ch := make(chan QuestEvent, 10)
	s.sseMu.Lock()
	s.sseClients[ch] = struct{}{}
	s.sseMu.Unlock()
	return ch
}

// Unsubscribe removes the channel from the broadcaster and closes it.
func (s *Service) Unsubscribe(ch chan QuestEvent) {
	s.sseMu.Lock()
	delete(s.sseClients, ch)
	close(ch)
	s.sseMu.Unlock()
}

// broadcastEvent sends an event to all connected SSE clients.
// Slow clients are skipped (non-blocking send).
func (s *Service) broadcastEvent(event QuestEvent) {
	s.sseMu.RLock()
	defer s.sseMu.RUnlock()

	if len(s.sseClients) == 0 {
		return
	}

	for ch := range s.sseClients {
		select {
		case ch <- event:
		default: // skip slow client
		}
	}
}

func (s *Service) broadcastSync(stats *SyncStats) {
	s.broadcastEvent(QuestEvent{Type: "sync_completed", Stats: stats})
}

// BroadcastStocksChanged notifies SSE clients that stock inventory has changed.
// Called by StockService via callback wired in the DI container.
func (s *Service) BroadcastStocksChanged(locationID int, action string) {
	s.broadcastEvent(QuestEvent{Type: "stocks_changed", LocationID: locationID, Action: action})
}

// SetTransferCreator sets the transfer creator (called after DI wiring to avoid circular deps)
func (s *Service) SetTransferCreator(tc TransferCreator) {
	s.transferCreator = tc
}

// SyncQuests fetches sheet data and aggregates into quests
func (s *Service) SyncQuests(ctx context.Context) ([]Quest, error) {
	// 1. Fetch categories (cache for matching)
	if err := s.loadCategories(); err != nil {
		return nil, fmt.Errorf("failed to load categories: %w", err)
	}

	// 2. Fetch sheet config from DB (fallback to env)
	sheetID, sheetName, err := s.getSheetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sheet config: %w", err)
	}

	// 3. Fetch sheet data
	rows, err := s.sheetReader.FetchSheet(sheetID, sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sheet: %w", err)
	}

	if len(rows) < 2 {
		return []Quest{}, nil // Empty or header-only sheet
	}

	// 4. Parse rows
	mapper := NewColumnMapper(rows[0])
	if !mapper.HasRequiredColumns() {
		return nil, fmt.Errorf("missing required columns: %v", mapper.MissingColumns())
	}

	sheetRows := []SheetRow{}
	consecutiveEmpty := 0
	for i := 1; i < len(rows); i++ {
		if isEmptyRow(rows[i]) {
			consecutiveEmpty++
			if consecutiveEmpty >= 3 {
				break
			}
			continue
		}
		consecutiveEmpty = 0
		sr, err := mapper.ParseRow(rows[i], i+1)
		if err != nil {
			log.Printf("[equipment-requests] Skipping row %d: %v", i+1, err)
			continue
		}
		sheetRows = append(sheetRows, *sr)
	}

	// 5. Match categories
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
		// Generate quest key
		key := s.questKey(row)

		// Get or create quest
		quest, exists := questMap[key]
		if !exists {
			quest = &Quest{
				ID:       generateQuestID(key),
				QuestKey: key,
				Destination: Destination{
					Pavilion: row.Pavilion,
					Location: row.Location,
				},
				Recipient:    row.Recipient,
				DeliveryDate: row.DeliveryDate,
				PickupTime:   row.PickupTime,
				BudgetOwner:  row.BudgetOwner,
				Status:       sheetStatusToQuestStatus(row.Status),
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

func sheetStatusToQuestStatus(sheetStatus string) string {
	switch sheetStatus {
	case StatusSent, StatusDelivered:
		return "completed"
	default:
		return "pending"
	}
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
	Deleted      int
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

	// 2. Fetch sheet config from DB (fallback to env)
	sheetID, sheetName, err := s.getSheetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sheet config: %w", err)
	}

	// 3. Fetch sheet data
	rows, err := s.sheetReader.FetchSheet(sheetID, sheetName)
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
	consecutiveEmpty2 := 0
	for i := 1; i < len(rows); i++ {
		if isEmptyRow(rows[i]) {
			consecutiveEmpty2++
			if consecutiveEmpty2 >= 3 {
				break
			}
			continue
		}
		consecutiveEmpty2 = 0
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
		if err := s.upsertQuest(ctx, &quests[i], stats); err != nil {
			log.Printf("[equipment-requests] Failed to upsert quest %s: %v", quests[i].ID, err)
			continue
		}
	}

	// 6b. Reconcile: drop stale pending quests no longer present in the sheet and not
	// linked to a transfer. This is what stops duplicates from a changed quest_key (e.g.
	// an edited pickup time) lingering forever. Safe because we already returned early on
	// an empty fetch (< 2 rows) and the repo no-ops when there are no keys to keep.
	keepKeys := make([]string, len(quests))
	for i := range quests {
		keepKeys[i] = quests[i].QuestKey
	}
	if deleted, err := s.questRepo.DeleteOrphanedPendingQuests(ctx, keepKeys); err != nil {
		log.Printf("[equipment-requests] Reconciliation failed: %v", err)
	} else if deleted > 0 {
		stats.Deleted = deleted
		log.Printf("[equipment-requests] Reconciliation removed %d stale pending quest(s)", deleted)
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
		SheetID:         sheetID,
	}

	if err := s.questRepo.CreateSyncLog(ctx, syncLog); err != nil {
		log.Printf("[equipment-requests] Failed to create sync log: %v", err)
	}

	log.Printf("[equipment-requests] Synced %d quests from %d rows (created: %d, updated: %d) in %dms",
		len(quests), len(sheetRows), stats.Created, stats.Updated, duration.Milliseconds())

	// Notify SSE clients
	go s.broadcastSync(stats)

	return &SyncResult{Quests: quests, Stats: stats}, nil
}

// upsertQuest creates or updates a quest in the database
func (s *Service) upsertQuest(ctx context.Context, quest *Quest, stats *SyncStats) error {
	// Try to get existing quest by key
	existing, err := s.questRepo.GetQuestByKey(ctx, quest.QuestKey)

	if err != nil || existing == nil {
		// Resolve location before create
		s.resolveAndSetQuestLocation(ctx, quest)
		if err := s.questRepo.CreateQuest(ctx, quest); err != nil {
			return fmt.Errorf("failed to create quest: %w", err)
		}
		stats.Created++
		stats.ItemsAdded += len(quest.Items)
		log.Printf("[equipment-requests] Created new quest: %s", quest.ID)
	} else {
		// Skip update for quests with active transfers or already completed/cancelled —
		// their status is managed externally.
		if existing.Status == "in_progress" || existing.Status == "completed" || existing.Status == "cancelled" {
			stats.Unchanged++
			log.Printf("[equipment-requests] Skipping update for quest %s — status is '%s'", existing.ID, existing.Status)
			return nil
		}

		// Skip update when sheet content is identical to what's already stored.
		if !questContentChanged(existing, quest) {
			stats.Unchanged++
			return nil
		}

		// Resolve location and set on quest before update
		s.resolveAndSetQuestLocation(ctx, quest)
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

// questContentChanged returns true if the incoming sheet data differs from what
// is already persisted — used to avoid unnecessary UpdateQuest calls on every sync.
func questContentChanged(existing, incoming *Quest) bool {
	if existing.Recipient != incoming.Recipient ||
		existing.DeliveryDate != incoming.DeliveryDate ||
		existing.Destination.Pavilion != incoming.Destination.Pavilion ||
		existing.Destination.Location != incoming.Destination.Location ||
		len(existing.Items) != len(incoming.Items) {
		return true
	}
	for i := range incoming.Items {
		if existing.Items[i].Name != incoming.Items[i].Name ||
			!intPtrEqual(existing.Items[i].Quantity, incoming.Items[i].Quantity) {
			return true
		}
	}
	return false
}

// intPtrEqual compares two optional ints by value (both nil == equal).
func intPtrEqual(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func (s *Service) resolveAndSetQuestLocation(ctx context.Context, quest *Quest) {
	id, matchType, _ := s.ResolveQuestLocationWithMatchType(quest)
	if id != nil {
		quest.LocationID = id
		quest.LocationResolved = true
		log.Printf("[equipment-requests] Resolved location for quest %s: location_id=%d (match=%s)", quest.ID, *id, matchType)
	} else {
		quest.LocationID = nil
		quest.LocationResolved = false
	}
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

// ============================================================================
// Phase 4: Quest → Transfer Integration
// ============================================================================

// ResolveQuestLocation resolves quest destination pavilion+location to a location ID
// using multi-strategy matching: manual mapping → exact → normalized → name-only.
func (s *Service) ResolveQuestLocation(quest *Quest) (*int, error) {
	id, _, err := s.ResolveQuestLocationWithMatchType(quest)
	return id, err
}

// ResolveQuestLocationWithMatchType returns location ID and match type (manual, exact, normalized, name_only, none).
func (s *Service) ResolveQuestLocationWithMatchType(quest *Quest) (*int, string, error) {
	pav := strings.TrimSpace(quest.Destination.Pavilion)
	loc := strings.TrimSpace(quest.Destination.Location)

	if pav == "" || loc == "" {
		return nil, "none", nil
	}

	ctx := context.Background()

	// 1. Manual mapping (highest priority)
	if id, err := s.questRepo.GetLocationMapping(ctx, pav, loc); err == nil && id != nil {
		_ = s.questRepo.IncrementLocationMappingUsage(ctx, pav, loc)
		return id, "manual", nil
	}

	// 2. Exact match — pavilion + name
	if id, err := s.questRepo.ResolveLocationByPavilionAndName(pav, loc); err == nil && id != nil {
		return id, "exact", nil
	}

	// 3. Normalized match — strip "Pawilon " prefix
	normalized := normalizePavilion(pav)
	if normalized != pav {
		if id, err := s.questRepo.ResolveLocationByPavilionAndName(normalized, loc); err == nil && id != nil {
			return id, "normalized", nil
		}
	}

	// 4. Name-only fallback (if exactly one location matches)
	if id, err := s.questRepo.ResolveLocationByNameOnly(loc); err == nil && id != nil {
		return id, "name_only", nil
	}

	return nil, "none", nil
}

func normalizePavilion(pav string) string {
	stripped := strings.TrimPrefix(strings.ToLower(pav), "pawilon ")
	if stripped != strings.ToLower(pav) {
		return strings.TrimSpace(stripped)
	}
	return pav
}

// ResolveQuestStockItems maps quest items (by category_id) to actual stock items at a source location
func (s *Service) ResolveQuestStockItems(quest *Quest, fromLocationID int) ([]ResolvedStockItem, []UnresolvedItem) {
	var resolved []ResolvedStockItem
	var unresolved []UnresolvedItem

	for _, item := range quest.Items {
		if item.Quantity == nil {
			unresolved = append(unresolved, UnresolvedItem{
				ItemName:   item.Name,
				Quantity:   nil,
				CategoryID: item.CategoryID,
				Reason:     "quantity not specified in sheet",
			})
			continue
		}

		if item.CategoryID == nil {
			unresolved = append(unresolved, UnresolvedItem{
				ItemName: item.Name,
				Quantity: item.Quantity,
				Reason:   "no category match for this item",
			})
			continue
		}

		matches, err := s.questRepo.FindStockItemsByCategory(fromLocationID, *item.CategoryID)
		if err != nil || len(matches) == 0 {
			unresolved = append(unresolved, UnresolvedItem{
				ItemName:   item.Name,
				Quantity:   item.Quantity,
				CategoryID: item.CategoryID,
				Reason:     "no stock found at source location for this category",
			})
			continue
		}

		// Use the first matching stock item (same category at that location)
		match := matches[0]
		resolved = append(resolved, ResolvedStockItem{
			StockID:      match.StockID,
			CategoryID:   match.CategoryID,
			CategoryName: match.CategoryName,
			ItemName:     item.Name,
			Quantity:     *item.Quantity,
			Available:    match.Quantity,
		})
	}

	return resolved, unresolved
}

// PreviewTransferFromQuest builds a preview of what a transfer from this quest would look like
func (s *Service) PreviewTransferFromQuest(ctx context.Context, questID string, fromLocationID int) (*TransferPreview, error) {
	quest, err := s.questRepo.GetQuestByID(ctx, questID)
	if err != nil {
		return nil, fmt.Errorf("quest not found: %w", err)
	}

	preview := &TransferPreview{
		FromLocationID: fromLocationID,
	}

	// Resolve destination location
	toLocationID, err := s.ResolveQuestLocation(quest)
	if err == nil && toLocationID != nil {
		preview.ToLocationID = toLocationID
	}

	// Resolve stock items
	preview.ResolvedItems, preview.UnresolvedItems = s.ResolveQuestStockItems(quest, fromLocationID)

	return preview, nil
}

// CreateTransferFromQuest creates an inventory transfer from a quest
func (s *Service) CreateTransferFromQuest(ctx context.Context, questID string, req CreateTransferFromQuestRequest) (int, error) {
	if s.transferCreator == nil {
		return 0, fmt.Errorf("transfer service not configured")
	}

	// 1. Fetch and validate quest
	quest, err := s.questRepo.GetQuestByID(ctx, questID)
	if err != nil {
		return 0, fmt.Errorf("quest not found: %w", err)
	}

	if quest.Status == "completed" || quest.Status == "cancelled" {
		return 0, fmt.Errorf("cannot create transfer for quest with status '%s'", quest.Status)
	}

	// 2. Resolve destination location
	toLocationID := 0
	if req.ToLocationID != nil {
		toLocationID = *req.ToLocationID
	} else {
		resolved, err := s.ResolveQuestLocation(quest)
		if err != nil || resolved == nil {
			return 0, fmt.Errorf("could not resolve destination location from pavilion '%s' and location '%s' — provide to_location_id explicitly",
				quest.Destination.Pavilion, quest.Destination.Location)
		}
		toLocationID = *resolved
	}

	// 3. Build transfer request
	transferReq := models.TransferRequest{
		FromLocationID: req.FromLocationID,
		LocationID:     toLocationID,
	}

	// 3a. Stock items — only what the request explicitly lists. We do NOT auto-resolve
	// from the quest sheet, so callers can send assets-only, stock-only, or a combination.
	// (Use the transfer-preview endpoint to get suggested stock items to send back here.)
	for _, si := range req.StockItems {
		transferReq.StockItemCollection = append(transferReq.StockItemCollection, models.StockItemRequest{
			ID:       si.ID,
			Quantity: si.Quantity,
		})
	}

	// Must move at least something.
	if len(req.StockItems) == 0 && len(req.Assets) == 0 {
		return 0, fmt.Errorf("transfer must include at least one stock item or asset")
	}

	// 3b. Assets — optional
	for _, a := range req.Assets {
		transferReq.AssetItemCollection = append(transferReq.AssetItemCollection, models.AssetItemRequest{
			ID: a.ID,
		})
	}

	// 3c. Users — optional
	for _, u := range req.Users {
		transferReq.Users = append(transferReq.Users, models.TransferUser{
			UserID: u.ID,
		})
	}

	// 4. Create the transfer
	transferID, err := s.transferCreator.InitTransfer(transferReq, "in_transit")
	if err != nil {
		return 0, fmt.Errorf("failed to create transfer: %w", err)
	}

	// 5. Link quest to transfer
	if err := s.questRepo.AddTransferToQuest(ctx, questID, transferID); err != nil {
		log.Printf("[equipment-requests] ORPHANED TRANSFER %d: created but failed to link to quest %s — manual cleanup required: %v", transferID, questID, err)
		return transferID, fmt.Errorf("transfer created (ID: %d) but failed to link to quest: %w", transferID, err)
	}

	log.Printf("[equipment-requests] Created transfer %d from quest %s", transferID, questID)
	if s.onTransferDispatched != nil {
		go s.onTransferDispatched(transferID)
	}
	return transferID, nil
}

// OnTransferStatusChanged is called by the transfer service when a linked transfer changes status.
// Implements the TransferStatusCallback interface.
func (s *Service) OnTransferStatusChanged(transferID int, newStatus string) error {
	ctx := context.Background()

	quest, err := s.questRepo.GetQuestByTransferID(ctx, transferID)
	if err != nil {
		return fmt.Errorf("failed to find quest for transfer %d: %w", transferID, err)
	}
	if quest == nil {
		return nil // transfer not linked to any quest
	}

	switch newStatus {
	case "completed":
		// Check whether any other active transfers remain.
		active, err := s.questRepo.GetActiveTransfersForQuest(ctx, quest.ID)
		if err != nil {
			return fmt.Errorf("failed to check active transfers for quest %s: %w", quest.ID, err)
		}
		// The completed transfer still has status "completed" in the DB at this point,
		// so GetActiveTransfersForQuest won't include it — length 0 means this was the last one.
		if len(active) == 0 {
			if err := s.questRepo.UpdateQuestStatus(ctx, quest.ID, "completed"); err != nil {
				return fmt.Errorf("failed to complete quest %s: %w", quest.ID, err)
			}
			log.Printf("[equipment-requests] Quest %s completed — all transfers done", quest.ID)
		} else {
			log.Printf("[equipment-requests] Transfer %d completed, quest %s still has %d active transfer(s)", transferID, quest.ID, len(active))
		}
		if s.onTransferEnded != nil {
			go s.onTransferEnded(transferID)
		}

	case "cancelled":
		// RemoveTransferFromQuest handles resetting quest to pending when all transfers are gone.
		if err := s.questRepo.RemoveTransferFromQuest(ctx, transferID); err != nil {
			return fmt.Errorf("failed to remove cancelled transfer %d from quest %s: %w", transferID, quest.ID, err)
		}
		log.Printf("[equipment-requests] Transfer %d cancelled, removed from quest %s", transferID, quest.ID)
		if s.onTransferEnded != nil {
			go s.onTransferEnded(transferID)
		}
	}

	return nil
}

// isEmptyRow returns true when every cell in the row is blank — signals end of sheet data.
func isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

