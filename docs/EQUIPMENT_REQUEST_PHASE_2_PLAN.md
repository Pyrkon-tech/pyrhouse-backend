# Equipment Request Integration - Phase 2: Database Persistence

## Goal
Add database persistence for equipment request quests, track sync history, enable quest lifecycle management, and implement fuzzy category matching.

## Status
📋 **PLANNED** - Ready for implementation

---

## Database Schema Design

### Migration: `000XXX_equipment_requests.up.sql`

```sql
-- Main quests table
CREATE TABLE equipment_request_quests (
    id SERIAL PRIMARY KEY,
    quest_key VARCHAR(255) UNIQUE NOT NULL,  -- MD5 hash for deduplication
    quest_id VARCHAR(100) UNIQUE NOT NULL,   -- Human-readable ID (quest-abc123)

    -- Destination
    destination_pavilion VARCHAR(100) NOT NULL,
    destination_location VARCHAR(100) NOT NULL,

    -- Recipient & delivery
    recipient VARCHAR(255) NOT NULL,
    delivery_date DATE NOT NULL,
    pickup_time VARCHAR(50),  -- Flexible format: "17-18", "jutro 15", etc.

    -- Metadata
    budget_owner VARCHAR(255),
    status VARCHAR(50) DEFAULT 'pending' NOT NULL,  -- pending, in_progress, completed, cancelled
    transfer_id INTEGER REFERENCES transfers(id),   -- Link to created transfer (nullable)

    -- Tracking
    last_synced_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,

    -- Index for common queries
    CONSTRAINT valid_status CHECK (status IN ('pending', 'in_progress', 'completed', 'cancelled'))
);

-- Quest items (many-to-one with quests)
CREATE TABLE equipment_request_items (
    id SERIAL PRIMARY KEY,
    quest_id INTEGER REFERENCES equipment_request_quests(id) ON DELETE CASCADE,

    -- Item details
    item_name VARCHAR(255) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),

    -- Category matching
    category_id INTEGER REFERENCES item_category(id),  -- Nullable if no match
    category_match_type VARCHAR(50) NOT NULL,  -- exact, fuzzy, manual, none
    category_match_confidence DECIMAL(3,2),    -- 0.00 to 1.00 for fuzzy matching

    -- Metadata
    budget_owner VARCHAR(255),  -- Per-item budget owner (can differ from quest)
    notes TEXT,
    source_row_number INTEGER,  -- Track back to spreadsheet row

    created_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT valid_match_type CHECK (category_match_type IN ('exact', 'fuzzy', 'manual', 'none'))
);

-- Sync history log
CREATE TABLE equipment_request_sync_log (
    id SERIAL PRIMARY KEY,
    synced_at TIMESTAMP DEFAULT NOW(),

    -- Stats
    rows_processed INTEGER NOT NULL,
    quests_created INTEGER NOT NULL DEFAULT 0,
    quests_updated INTEGER NOT NULL DEFAULT 0,
    quests_unchanged INTEGER NOT NULL DEFAULT 0,
    items_added INTEGER NOT NULL DEFAULT 0,
    items_removed INTEGER NOT NULL DEFAULT 0,

    -- Error tracking
    errors TEXT,  -- JSON array of errors if any
    success BOOLEAN DEFAULT true,

    -- Performance
    duration_ms INTEGER,  -- Sync duration in milliseconds
    sheet_id VARCHAR(255) NOT NULL
);

-- Manual category mapping overrides
CREATE TABLE equipment_request_category_mapping (
    id SERIAL PRIMARY KEY,
    form_item_name VARCHAR(255) UNIQUE NOT NULL,  -- Exact match from form
    category_id INTEGER REFERENCES item_category(id) ON DELETE CASCADE,

    -- Metadata
    created_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP,
    use_count INTEGER DEFAULT 0
);

-- Indexes for performance
CREATE INDEX idx_quests_status ON equipment_request_quests(status);
CREATE INDEX idx_quests_delivery_date ON equipment_request_quests(delivery_date);
CREATE INDEX idx_quests_last_synced ON equipment_request_quests(last_synced_at);
CREATE INDEX idx_quests_quest_key ON equipment_request_quests(quest_key);
CREATE INDEX idx_items_quest_id ON equipment_request_items(quest_id);
CREATE INDEX idx_items_category_id ON equipment_request_items(category_id);
CREATE INDEX idx_sync_log_synced_at ON equipment_request_sync_log(synced_at DESC);
```

### Migration: `000XXX_equipment_requests.down.sql`

```sql
DROP TABLE IF EXISTS equipment_request_category_mapping CASCADE;
DROP TABLE IF EXISTS equipment_request_sync_log CASCADE;
DROP TABLE IF EXISTS equipment_request_items CASCADE;
DROP TABLE IF EXISTS equipment_request_quests CASCADE;
```

---

## Implementation Tasks

### Task 1: Database Migrations
**File:** `migrations/000XXX_equipment_requests.{up,down}.sql`

- [ ] Create migration files with next sequential number
- [ ] Test up migration locally
- [ ] Test down migration (rollback)
- [ ] Verify indexes are created
- [ ] Check foreign key constraints work

**Acceptance:**
- ✅ `go run main.go -migrate` succeeds
- ✅ `psql` shows all tables and indexes
- ✅ Down migration removes all tables cleanly

---

### Task 2: Repository Layer
**File:** `internal/equipment_requests/repository.go`

```go
package equipment_requests

import (
	"context"
	"warehouse/internal/models"
	"warehouse/internal/repository"
	"github.com/doug-martin/goqu/v9"
)

type Repository struct {
	repo *repository.Repository
}

func NewRepository(repo *repository.Repository) *Repository {
	return &Repository{repo: repo}
}

// Quest CRUD
func (r *Repository) CreateQuest(ctx context.Context, quest *Quest) error
func (r *Repository) UpdateQuest(ctx context.Context, quest *Quest) error
func (r *Repository) GetQuestByID(ctx context.Context, questID string) (*Quest, error)
func (r *Repository) GetQuestByKey(ctx context.Context, questKey string) (*Quest, error)
func (r *Repository) ListQuests(ctx context.Context, filter QuestFilter) ([]Quest, error)
func (r *Repository) DeleteQuest(ctx context.Context, questID string) error

// Quest status management
func (r *Repository) UpdateQuestStatus(ctx context.Context, questID string, status string) error
func (r *Repository) LinkQuestToTransfer(ctx context.Context, questID string, transferID int) error

// Item management
func (r *Repository) CreateItems(ctx context.Context, questID int, items []QuestItem) error
func (r *Repository) DeleteItemsByQuestID(ctx context.Context, questID int) error

// Sync log
func (r *Repository) CreateSyncLog(ctx context.Context, log *SyncLog) error
func (r *Repository) GetLatestSyncLog(ctx context.Context) (*SyncLog, error)

// Category mapping
func (r *Repository) GetCategoryMapping(ctx context.Context, itemName string) (*int, error)
func (r *Repository) CreateCategoryMapping(ctx context.Context, mapping *CategoryMapping) error
func (r *Repository) IncrementMappingUsage(ctx context.Context, itemName string) error
```

**Implementation Details:**
- Use `goqu` for all queries (no raw SQL)
- Use transactions for multi-table operations (quest + items creation)
- Handle unique constraint violations gracefully (quest_key duplicates)
- Return `apperrors.UniqueViolationError` for duplicates

**Acceptance:**
- ✅ All CRUD operations work
- ✅ Transactions rollback on error
- ✅ Indexes are used (check with EXPLAIN)

---

### Task 3: Service Updates
**File:** `internal/equipment_requests/service.go`

**Add:**
```go
type Service struct {
	sheetReader      *googlesheets.DutyScheduleService
	categoryRepo     *category.CategoryRepository
	questRepo        *Repository  // NEW
	sheetID          string
	sheetName        string
	fuzzyThreshold   int
	categories       []models.ItemCategory
}

// SyncQuestsToDatabase replaces SyncQuests
func (s *Service) SyncQuestsToDatabase(ctx context.Context) (*SyncResult, error) {
	// 1. Fetch from sheets (same as Phase 1)
	rows, err := s.sheetReader.FetchSheet(s.sheetID, s.sheetName)

	// 2. Parse and aggregate (same as Phase 1)
	quests := s.aggregateQuests(rows)

	// 3. Match categories (IMPROVED with fuzzy matching)
	for i := range quests {
		for j := range quests[i].Items {
			match := s.matchCategoryWithFuzzy(quests[i].Items[j].Name)
			quests[i].Items[j].CategoryID = match.CategoryID
			quests[i].Items[j].CategoryMatch = match.MatchType
		}
	}

	// 4. Upsert to database
	stats := &SyncStats{}
	for _, quest := range quests {
		err := s.upsertQuest(ctx, &quest, stats)
		if err != nil {
			return nil, err
		}
	}

	// 5. Log sync result
	syncLog := &SyncLog{
		RowsProcessed:   len(rows),
		QuestsCreated:   stats.Created,
		QuestsUpdated:   stats.Updated,
		ItemsAdded:      stats.ItemsAdded,
		Success:         true,
	}
	s.questRepo.CreateSyncLog(ctx, syncLog)

	return &SyncResult{Quests: quests, Stats: stats}, nil
}

func (s *Service) upsertQuest(ctx context.Context, quest *Quest, stats *SyncStats) error {
	// Check if quest exists by quest_key
	existing, err := s.questRepo.GetQuestByKey(ctx, quest.questKey(quest))

	if err != nil {
		// Create new quest
		err = s.questRepo.CreateQuest(ctx, quest)
		stats.Created++
	} else {
		// Update existing quest
		quest.ID = existing.ID
		err = s.questRepo.UpdateQuest(ctx, quest)
		stats.Updated++
	}

	return err
}

// Fuzzy category matching (NEW)
func (s *Service) matchCategoryWithFuzzy(itemName string) CategoryMatch {
	// 1. Check manual mapping first
	if categoryID, err := s.questRepo.GetCategoryMapping(ctx, itemName); err == nil {
		s.questRepo.IncrementMappingUsage(ctx, itemName)
		return CategoryMatch{
			CategoryID: categoryID,
			MatchType:  "manual",
			Confidence: 1.0,
		}
	}

	// 2. Exact match
	for _, cat := range s.categories {
		if cat.Name == itemName || cat.Label == itemName {
			return CategoryMatch{
				CategoryID: &cat.ID,
				MatchType:  "exact",
				Confidence: 1.0,
			}
		}
	}

	// 3. Fuzzy match (Levenshtein distance)
	bestMatch := s.findBestFuzzyMatch(itemName)
	if bestMatch.Distance <= s.fuzzyThreshold {
		return CategoryMatch{
			CategoryID: &bestMatch.CategoryID,
			MatchType:  "fuzzy",
			Confidence: 1.0 - float64(bestMatch.Distance)/float64(len(itemName)),
		}
	}

	// 4. No match
	return CategoryMatch{
		MatchType:  "none",
		Confidence: 0.0,
	}
}

// Levenshtein distance implementation
func (s *Service) findBestFuzzyMatch(itemName string) FuzzyMatchResult {
	// TODO: Implement Levenshtein distance algorithm
	// Compare itemName against all categories
	// Return best match with distance <= threshold
}
```

**Acceptance:**
- ✅ Sync creates new quests in DB
- ✅ Sync updates existing quests (by quest_key)
- ✅ Fuzzy matching works (test with "Laptoop" → "Laptop")
- ✅ Manual mappings override fuzzy matching
- ✅ Sync log is created after each sync

---

### Task 4: Handler Updates
**File:** `internal/equipment_requests/handler.go`

**Replace in-memory cache with DB queries:**

```go
type Handler struct {
	service *Service
	// Remove: quests []Quest  (no more in-memory cache)
}

// ManualSync now persists to DB
func (h *Handler) ManualSync(c *gin.Context) {
	result, err := h.service.SyncQuestsToDatabase(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to sync equipment requests",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Sync completed successfully",
		"stats": gin.H{
			"quests_created": result.Stats.Created,
			"quests_updated": result.Stats.Updated,
			"items_added":    result.Stats.ItemsAdded,
		},
		"quests": result.Quests,
	})
}

// ListQuests now queries DB
func (h *Handler) ListQuests(c *gin.Context) {
	filter := QuestFilter{
		Status: c.Query("status"),  // Filter by status
		Limit:  getIntQuery(c, "limit", 100),
		Offset: getIntQuery(c, "offset", 0),
	}

	quests, err := h.service.questRepo.ListQuests(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(quests),
		"quests": quests,
	})
}

// GetQuest now queries DB
func (h *Handler) GetQuest(c *gin.Context) {
	questID := c.Param("id")

	quest, err := h.service.questRepo.GetQuestByID(c.Request.Context(), questID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quest not found"})
		return
	}

	c.JSON(http.StatusOK, quest)
}

// NEW: Update quest status
func (h *Handler) UpdateQuestStatus(c *gin.Context) {
	questID := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := h.service.questRepo.UpdateQuestStatus(c.Request.Context(), questID, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
}

// NEW: Create transfer from quest
func (h *Handler) CreateTransferFromQuest(c *gin.Context) {
	questID := c.Param("id")

	// 1. Fetch quest
	quest, err := h.service.questRepo.GetQuestByID(c.Request.Context(), questID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quest not found"})
		return
	}

	// 2. Build transfer payload
	transferPayload := buildTransferFromQuest(quest)

	// 3. Create transfer (reuse existing transfer creation logic)
	// ... (call transfer service or return payload for frontend)

	c.JSON(http.StatusOK, gin.H{
		"message": "Transfer created",
		"transfer_payload": transferPayload,
	})
}
```

**New endpoints to add:**
- `PATCH /api/equipment-requests/quests/:id/status` - Update quest status
- `POST /api/equipment-requests/quests/:id/transfer` - Create transfer from quest
- `GET /api/equipment-requests/sync-log` - View sync history
- `POST /api/equipment-requests/category-mapping` - Add manual category mapping

**Acceptance:**
- ✅ All endpoints use DB instead of in-memory cache
- ✅ Quest status can be updated
- ✅ Transfer creation works
- ✅ Sync log is viewable

---

### Task 5: Update DI Container
**File:** `internal/di/container.go`

```go
var equipmentRequestHandler *equipment_requests.Handler
if cfg.EquipmentRequest.SheetID != "" && googleSheetsHandler != nil {
	categoryRepo := category.NewCategoryRepository(repo)
	equipmentRequestRepo := equipment_requests.NewRepository(repo)  // NEW
	equipmentRequestService := equipment_requests.NewService(
		googleSheetsHandler.DutyScheduleService,
		categoryRepo,
		equipmentRequestRepo,  // NEW
		cfg.EquipmentRequest.SheetID,
		cfg.EquipmentRequest.SheetName,
		cfg.EquipmentRequest.FuzzyThreshold,
	)
	equipmentRequestHandler = equipment_requests.NewHandler(equipmentRequestService)
}
```

---

### Task 6: Testing

**Unit Tests:**
- `repository_test.go` - Test all CRUD operations with test DB
- `service_test.go` - Update existing tests + add fuzzy matching tests
- `handler_test.go` - HTTP handler tests with mocked service

**Integration Tests:**
- Full sync flow: Sheet → DB → Query
- Quest lifecycle: pending → in_progress → completed
- Transfer creation from quest

**Test Coverage Goal:** 70%+

---

### Task 7: Documentation Updates

1. **OpenAPI spec** (`docs/openapi.yaml`):
   - Add new endpoints (PATCH status, POST transfer, GET sync-log, POST category-mapping)
   - Update existing endpoints to show DB-backed behavior
   - Add filter parameters (status, limit, offset)

2. **AGENTS.md**:
   - Update equipment_requests example to show DB usage
   - Document fuzzy matching algorithm

3. **Implementation Plan**:
   - Mark Phase 2 as completed
   - Add performance benchmarks

---

## Data Migration Strategy

### Option A: Fresh Start (Recommended for Phase 2)
- No migration needed - Phase 1 was in-memory only
- First sync populates DB from scratch
- Clean slate

### Option B: If Production Data Exists
Not applicable for Phase 2 (Phase 1 has no persistent data)

---

## Rollout Plan

### Step 1: Deploy Database Schema
```bash
# On production
git pull origin main
go run main.go -migrate
```

### Step 2: Enable Phase 2 in Config
```env
# No changes needed - automatic based on code
```

### Step 3: Run Initial Sync
```bash
curl -X POST https://api.pyrhouse.space/api/equipment-requests/sync \
  -H "Authorization: Bearer $TOKEN"
```

### Step 4: Verify Data
```bash
# Check quest count
curl https://api.pyrhouse.space/api/equipment-requests/quests \
  -H "Authorization: Bearer $TOKEN"
```

---

## Success Criteria

- ✅ All migrations run successfully
- ✅ Repository layer fully tested (70%+ coverage)
- ✅ Sync creates/updates quests in DB
- ✅ Fuzzy category matching works with >80% accuracy
- ✅ Manual category mappings override fuzzy matching
- ✅ Quest lifecycle management works (status updates)
- ✅ Sync log tracks all syncs
- ✅ All endpoints use DB (no in-memory cache)
- ✅ OpenAPI docs updated
- ✅ No breaking changes to Phase 1 API contract

---

## Estimated Effort

- **Migrations:** 1 hour
- **Repository:** 4 hours
- **Service updates:** 3 hours
- **Handler updates:** 2 hours
- **Testing:** 4 hours
- **Documentation:** 1 hour

**Total:** ~15 hours (2 days)

---

## Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Migration fails in production | High | Test thoroughly locally, have rollback plan |
| Fuzzy matching too aggressive | Medium | Use conservative threshold (3), allow manual overrides |
| Performance issues with large sheets | Medium | Add indexes, pagination, consider caching |
| Data inconsistency between sheet and DB | Low | Sync log tracks all changes, can re-sync |

---

## Next Steps After Phase 2

**Phase 3:** Auto-sync scheduler (in-process goroutine)
- Enable `EQUIPMENT_REQUEST_SYNC_ENABLED=true`
- Add graceful shutdown handling
- Add sync status endpoint

**Phase 4:** Frontend integration + SSE for real-time updates
