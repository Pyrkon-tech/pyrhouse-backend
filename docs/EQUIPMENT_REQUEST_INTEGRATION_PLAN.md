# Equipment Request Form Integration — Implementation Plan

## Overview

Integration with Google Sheets equipment request form to automatically aggregate orders into equipment release "quests" (transfers).

**Sheet:** [Zamówienia](https://docs.google.com/spreadsheets/d/16BytrbWmyWeBGnlSIDZn1Lnb5rdspoQu_rpc5m5Vtbc/edit?usp=sharing)

## Business Requirements

### Data Model (from spreadsheet)

**Sheet: "Zamówienia"**

| Column | Polish Name | Type | Purpose | Grouping Field? |
|--------|-------------|------|---------|-----------------|
| Item | Rzeczy | String | Equipment type/name | ✓ (if matches category) |
| Quantity | Ilość | Integer | Amount requested | ✓ (sum per item) |
| Pavilion | Pawilon | String | Destination pavilion | ✓ |
| Location | Miejsce | String | Destination location | ✓ |
| Status | Stan | String | Current status (Zamówione/Dostarczone/Wysłane) | Filter field |
| Pickup Time | Godzina odbioru | String | When to pick up (flexible: "17-18", "jutro 15", "17") | ✓ (optional) |
| Delivery Date | Dostawa do | Date | When to deliver | ✓ |
| Budget Owner | Osoba odpowiedzialna za budżet | String | Budget approval contact | ✗ (metadata only) |
| Recipient | Do kogo ma trafić | String | Final recipient | ✓ |
| Notes | UWAGI | Text | Additional info | ✗ (metadata) |

### Aggregation Logic ("Quest" Creation)

**Grouping Key:**
```
quest_key = f"{pavilion}|{location}|{recipient}|{delivery_date}|{pickup_time}"
```

**Quest Structure:**
```json
{
  "quest_id": "auto-generated-uuid",
  "destination": {
    "pavilion": "Pawilon 6",
    "location": "Magazyn Techniczny"
  },
  "recipient": "Karol Taki",
  "delivery_date": "2025-06-10",
  "pickup_time": null,
  "items": [
    {
      "name": "Drukarka A4",
      "quantity": 1,
      "category_id": 123,  // if matched
      "category_match": "exact|fuzzy|none",
      "budget_owner": "Czesław Jakiś",
      "notes": ""
    }
  ],
  "source_rows": [1, 2, 5],  // spreadsheet row numbers
  "last_synced": "2025-02-15T10:30:00Z",
  "status": "pending|in_progress|completed"
}
```

### Key Challenges & Solutions

1. **Item Category Matching**
   - **Challenge:** Form "Rzeczy" may not match existing `item_category.name`
   - **Solution:**
     - Fuzzy matching (Levenshtein distance < 3)
     - Manual mapping table in DB (form_name → category_id)
     - Fallback: present raw form data, allow frontend selection

2. **Schema Flexibility**
   - **Challenge:** Spreadsheet columns may change
   - **Solution:**
     - **Column mapping hardcoded in Go** (`column_mapper.go`)
     - Easy to update, type-safe
     - Can migrate to DB later if needed
     - Header auto-detection on first sync
     - Graceful degradation if expected columns missing

3. **Incremental Updates**
   - **Challenge:** Avoid re-processing entire sheet every time
   - **Solution:**
     - Track last processed row number
     - Check for new rows only
     - Handle edits: re-sync if sheet modified date changed

---

## Architecture Design (SOLID)

### Module: `internal/equipment_requests/`

```
internal/equipment_requests/
├── models.go           # Quest, QuestItem, SheetRow structs
├── service.go          # Business logic: aggregation, matching
├── repository.go       # Optional: store quests in DB (Phase 2+)
├── handler.go          # HTTP endpoints
├── sync_scheduler.go   # Periodic sync job
└── column_mapper.go    # Flexible column mapping
```

### Dependencies

```
Google Sheets API
      ↓
SheetReader (existing: internal/integrations/googlesheets/)
      ↓
EquipmentRequestService
      ↓
    ┌─────┴─────┐
    ↓           ↓
CategoryMatcher  QuestAggregator
```

---

## Implementation Phases

### Phase 1: Core Sync & Aggregation (No DB, In-Memory) ✅ **COMPLETED**

**Goal:** Prove the concept, manual trigger via API

**Status:** ✅ Implemented, tested, and working (2026-02-15)

**Completed Tasks:**
1. ✅ **Reused existing Google Sheets client** (`internal/integrations/googlesheets/`)
   - Added `FetchSheet()` method to `DutyScheduleService`
   - Exported `DutyScheduleService` field in handler for access
2. ✅ **Created `EquipmentRequestService`**
   - `SyncQuests(ctx)` → fetches and returns `[]Quest`
   - Groups rows by quest key (pavilion + location + recipient + delivery_date + pickup_time)
   - Exact category matching implemented (fuzzy matching TODO in Phase 2)
3. ✅ **Added HTTP endpoints:**
   - `POST /api/equipment-requests/sync` - Manual trigger, returns quests
   - `GET /api/equipment-requests/quests` - List cached quests
   - `GET /api/equipment-requests/quests/:id` - Get single quest
4. ✅ **Configuration** (`.env`):
   ```env
   EQUIPMENT_REQUEST_SHEET_ID=16BytrbWmyWeBGnlSIDZn1Lnb5rdspoQu_rpc5m5Vtbc
   EQUIPMENT_REQUEST_SHEET_NAME=Zamówienia
   EQUIPMENT_REQUEST_SYNC_ENABLED=false  # Manual only in Phase 1
   EQUIPMENT_REQUEST_FUZZY_THRESHOLD=3
   ```
5. ✅ **Column Mapping** (`column_mapper.go`)
   - Hardcoded Polish → English field mapping
   - Flexible header detection
   - Validation of required columns
6. ✅ **Test Coverage**
   - `column_mapper_test.go`: Header parsing, row validation
   - `service_test.go`: Quest aggregation, key generation, filtering
   - All tests passing (9 test cases, 0 failures)
7. ✅ **API Documentation** (`docs/openapi.yaml`)
   - Added `equipment-requests` tag
   - Added 3 endpoint specifications (POST sync, GET quests, GET quest by ID)
   - Added schemas: `EquipmentRequestQuest`, `QuestDestination`, `QuestItem`
   - Added `Unauthorized` response for protected endpoints

**Deliverables:**
- ✅ Manual sync works (944ms for 129 rows)
- ✅ Returns aggregated quests (2 quests from test sheet)
- ✅ Frontend can consume JSON
- ✅ In-memory cache for fast retrieval (<150µs)
- ✅ Status filtering (only "Zamówione" items)
- ✅ Source row tracking for debugging

**Performance:**
- Sync time: ~944ms for 129 rows
- Quest retrieval: <150µs (in-memory)
- Test results: 2 quests aggregated, 1 invalid row skipped

---

### Phase 2: Persistence & State Management (DB)

**Goal:** Track sync history, quest lifecycle

**Database Schema:**

```sql
-- Table: equipment_request_quests
CREATE TABLE equipment_request_quests (
    id SERIAL PRIMARY KEY,
    quest_key VARCHAR(255) UNIQUE NOT NULL,  -- hash of grouping fields
    destination_pavilion VARCHAR(100),
    destination_location VARCHAR(100),
    recipient VARCHAR(255),
    delivery_date DATE,
    pickup_time TIME,
    status VARCHAR(50) DEFAULT 'pending',  -- pending, in_progress, completed
    transfer_id INTEGER REFERENCES transfers(id),  -- link to created transfer
    last_synced_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Table: equipment_request_items
CREATE TABLE equipment_request_items (
    id SERIAL PRIMARY KEY,
    quest_id INTEGER REFERENCES equipment_request_quests(id) ON DELETE CASCADE,
    item_name VARCHAR(255) NOT NULL,
    quantity INTEGER NOT NULL,
    category_id INTEGER REFERENCES item_categories(id),  -- nullable
    category_match_type VARCHAR(50),  -- exact, fuzzy, none
    budget_owner VARCHAR(255),
    notes TEXT,
    source_row_number INTEGER  -- for tracking back to sheet
);

-- Table: equipment_request_sync_log
CREATE TABLE equipment_request_sync_log (
    id SERIAL PRIMARY KEY,
    synced_at TIMESTAMP DEFAULT NOW(),
    rows_processed INTEGER,
    quests_created INTEGER,
    quests_updated INTEGER,
    errors TEXT
);

-- Table: equipment_request_category_mapping (manual overrides)
CREATE TABLE equipment_request_category_mapping (
    id SERIAL PRIMARY KEY,
    form_item_name VARCHAR(255) UNIQUE NOT NULL,
    category_id INTEGER REFERENCES item_categories(id),
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Tasks:**
1. Create migrations
2. Update service to use repository
3. Track quest lifecycle
4. Add `POST /api/quests/:id/create-transfer` endpoint

---

### Phase 3: Automated Sync (Cron/Scheduler)

**Goal:** Auto-sync every X minutes, cheapest solution for DigitalOcean droplet

**Recommended: In-Process Goroutine with time.Ticker** ✅ CHEAPEST
- **Cost:** Zero extra resources, ~0 CPU when idle
- **Simplicity:** No dependencies, no system cron setup
- **Reliability:** Runs as part of main process, automatic restart with app
- **Memory:** ~100KB for goroutine

```go
// Simple implementation in main.go
func startEquipmentRequestSync(service *equipment_requests.Service) {
    interval := 15 * time.Minute
    ticker := time.NewTicker(interval)

    go func() {
        for range ticker.C {
            if err := service.SyncQuests(context.Background()); err != nil {
                log.Printf("Equipment request sync failed: %v", err)
            }
        }
    }()
}
```

**Alternative: robfig/cron** (if you need complex schedules)
- Adds dependency but more flexible cron expressions
- Still in-process, same cost as Ticker

**NOT Recommended for Droplet:**
- ❌ External cron (requires system access, harder to deploy)
- ❌ Separate worker process (doubles memory usage)
- ❌ Message queue (overkill, adds Redis/RabbitMQ costs)

**Configuration:**
```env
EQUIPMENT_REQUEST_SYNC_ENABLED=true
EQUIPMENT_REQUEST_SYNC_INTERVAL=15m  # parsed by time.ParseDuration
```

**Tasks:**
1. Add simple ticker in `main.go` or `sync_scheduler.go`
2. Graceful shutdown (stop ticker, finish current sync)
3. Add sync status endpoint: `GET /api/equipment-requests/sync-status`

---

### Phase 4: Advanced Features (Future)

1. **Frontend Integration** ⭐ Priority
   - Quest list view
   - "Create Transfer from Quest" button
   - Pre-filled transfer form with quest items
   - Quest status tracking

2. **Conflict Resolution**
   - Detect manual edits to sheet
   - Handle duplicate entries
   - Row deletion handling

3. **Real-time Updates** (Optional)
   - SSE (Server-Sent Events) for live quest updates
   - Frontend subscribes to `/api/equipment-requests/stream`
   - Push new quests without polling

4. **~~Notifications~~** (Skipped)
   - ~~Slack/Discord integration~~
   - ~~Email alerts~~
   - ~~Webhooks~~ (maybe SSE instead)

5. **~~Analytics~~** (Skipped for now)
   - ~~Quest completion rate~~
   - ~~Average processing time~~
   - ~~Budget owner breakdown~~

---

## API Endpoints (RESTful)

### Phase 1 (Manual Sync)
```
GET  /api/equipment-requests/sync         # Manual trigger, returns quests
GET  /api/equipment-requests/quests       # List all quests (in-memory)
```

### Phase 2 (Persistence)
```
GET  /api/equipment-requests/quests           # List quests (from DB)
GET  /api/equipment-requests/quests/:id       # Quest details
POST /api/equipment-requests/quests/:id/transfer  # Create transfer
PATCH /api/equipment-requests/quests/:id     # Update status
```

### Phase 3 (Auto-sync)
```
GET  /api/equipment-requests/sync-status  # Last sync time, next sync
POST /api/equipment-requests/sync/trigger # Force sync (admin)
```

### Phase 4 (Mappings)
```
GET  /api/equipment-requests/category-mappings
POST /api/equipment-requests/category-mappings
```

---

## Configuration (Environment Variables)

```env
# Google Sheets
EQUIPMENT_REQUEST_SHEET_ID=16BytrbWmyWeBGnlSIDZn1Lnb5rdspoQu_rpc5m5Vtbc
EQUIPMENT_REQUEST_SHEET_NAME=Zamówienia

# Sync Settings
EQUIPMENT_REQUEST_SYNC_ENABLED=true
EQUIPMENT_REQUEST_SYNC_INTERVAL=15m

# Matching
EQUIPMENT_REQUEST_FUZZY_THRESHOLD=3  # Levenshtein distance
```

**Column Mapping:** Hardcoded in `internal/equipment_requests/column_mapper.go`
- Type-safe, easy to update in code
- Can migrate to DB later if needed
- No env var pollution

---

## Code Structure

### Service Layer (SOLID Example)

```go
// service.go
package equipment_requests

type Service struct {
    sheetReader      *googlesheets.Service
    categoryMatcher  CategoryMatcher
    questAggregator  QuestAggregator
    repository       *Repository  // nil in Phase 1
}

func (s *Service) SyncQuests(ctx context.Context) ([]Quest, error) {
    // 1. Fetch sheet data
    rows, err := s.sheetReader.FetchSheet(sheetID, sheetName)
    if err != nil {
        return nil, err
    }

    // 2. Parse into structured data
    requests := s.parseRows(rows)

    // 3. Match categories
    for i := range requests {
        requests[i].CategoryMatch = s.categoryMatcher.Match(requests[i].ItemName)
    }

    // 4. Aggregate into quests
    quests := s.questAggregator.Aggregate(requests)

    // 5. Persist (Phase 2+)
    if s.repository != nil {
        quests, err = s.repository.UpsertQuests(quests)
    }

    return quests, err
}

// Single Responsibility: each method does one thing
func (s *Service) parseRows(rows [][]string) []EquipmentRequest {...}
```

### Category Matcher (Strategy Pattern)

```go
// column_mapper.go
type CategoryMatcher interface {
    Match(itemName string) CategoryMatch
}

type FuzzyCategoryMatcher struct {
    categories []models.ItemCategory
    threshold  int
}

func (m *FuzzyCategoryMatcher) Match(itemName string) CategoryMatch {
    // 1. Exact match
    for _, cat := range m.categories {
        if strings.EqualFold(cat.Name, itemName) {
            return CategoryMatch{
                CategoryID: cat.ID,
                Type: "exact",
                Confidence: 1.0,
            }
        }
    }

    // 2. Fuzzy match (Levenshtein)
    bestMatch := m.findBestMatch(itemName)
    if bestMatch.Distance <= m.threshold {
        return CategoryMatch{
            CategoryID: bestMatch.ID,
            Type: "fuzzy",
            Confidence: 1.0 - (float64(bestMatch.Distance) / 10.0),
        }
    }

    // 3. No match
    return CategoryMatch{Type: "none"}
}
```

---

## Testing Strategy

### Unit Tests
```go
func TestQuestAggregation(t *testing.T) {
    tests := []struct {
        name     string
        input    []EquipmentRequest
        expected []Quest
    }{
        {
            name: "groups by destination and recipient",
            input: []EquipmentRequest{
                {ItemName: "Laptop", Pavilion: "P6", Location: "Mag", Recipient: "Jan"},
                {ItemName: "Mouse", Pavilion: "P6", Location: "Mag", Recipient: "Jan"},
            },
            expected: []Quest{
                {
                    Destination: Destination{Pavilion: "P6", Location: "Mag"},
                    Recipient: "Jan",
                    Items: []QuestItem{
                        {Name: "Laptop", Quantity: 1},
                        {Name: "Mouse", Quantity: 1},
                    },
                },
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            aggregator := NewQuestAggregator()
            result := aggregator.Aggregate(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Integration Tests
- Mock Google Sheets API responses
- Test full sync flow
- Verify DB state after sync

---

## Migration Path

### From Current State → Phase 1

1. **No Breaking Changes:** Add new module alongside existing code
2. **Reuse:** Existing `googlesheets` integration
3. **API:** New endpoints under `/api/equipment-requests/*`

### From Phase 1 → Phase 2

1. **Migration:** Create DB tables
2. **Service Update:** Add repository injection
3. **Backward Compat:** Keep in-memory mode as fallback

### From Phase 2 → Phase 3

1. **Config:** Add scheduler settings
2. **Main:** Start scheduler goroutine
3. **Monitoring:** Add health check for scheduler

---

## Security Considerations

1. **Spreadsheet Access**
   - Service Account credentials in `GOOGLE_SHEETS_CREDENTIALS_JSON`
   - Read-only permissions
   - Sheet ID validation

2. **No PII Scanning**
   - Budget owner names not used for training
   - Recipient names treated as metadata only
   - Compliance: GDPR-aware data handling

3. **Rate Limiting**
   - Google Sheets API quota: 100 requests/100s
   - Internal sync throttling

---

## Success Metrics

### Phase 1
- ✅ Manual sync returns valid JSON
- ✅ Category matching >70% accuracy
- ✅ Aggregation groups correctly

### Phase 2
- ✅ Quests persisted to DB
- ✅ Create transfer from quest works
- ✅ No data loss during sync

### Phase 3
- ✅ Auto-sync runs every 15min
- ✅ No missed syncs for 7 days
- ✅ Sync latency <5s

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Sheet schema changes | High | Column mapping config, graceful degradation |
| API quota exceeded | Medium | Caching, exponential backoff |
| Category mismatch | Medium | Manual mapping table, frontend override |
| Duplicate entries | Low | Dedupe by quest_key, log warnings |
| Lost updates (race) | Low | Optimistic locking, last-write-wins |

---

## Next Steps

1. **Validate Plan:** Review with stakeholders
2. **Proof of Concept:** Implement Phase 1 (1-2 days)
3. **Frontend Mockup:** Design quest list UI
4. **DB Schema Review:** Finalize before Phase 2
5. **Iterate:** Gather feedback, adjust

---

## Implementation Notes

### Pickup Time Parsing (Flexible)

```go
// Examples from real data:
// "17-18"      → 17:00-18:00 range
// "jutro 15"   → tomorrow at 15:00
// "17"         → 17:00

func parsePickupTime(raw string) (*PickupTime, error) {
    // Detect range (e.g., "17-18")
    if strings.Contains(raw, "-") {
        parts := strings.Split(raw, "-")
        // Parse as range
    }

    // Detect relative time (e.g., "jutro")
    if strings.Contains(raw, "jutro") {
        // Parse tomorrow + time
    }

    // Parse single time (e.g., "17")
    // ...

    // Fallback: store raw string
    return &PickupTime{Raw: raw}, nil
}
```

### Status Mapping

```go
const (
    StatusOrdered    = "Zamówione"   // From form
    StatusDelivered  = "Dostarczone" // From form
    StatusSent       = "Wysłane"     // From form
    StatusReported   = "Zgłoszone"   // Future
)
```

### Budget Owner

- **Shared per quest** (all items in same quest have same budget owner)
- Stored as quest-level metadata, not item-level

### Notes Examples

Real-world complexity:
```
"dobrze, żeby łączyła się z laptopami bez składania gżdaczy w ofierze,
dostawa na antresolę w Iglicy"
```

→ Store as-is, display in frontend

---

## Cost Analysis (DigitalOcean Droplet)

### Current Droplet Specs (assumed)
- $6/month basic droplet: 1 vCPU, 1GB RAM

### Sync Cost (In-Process Goroutine)
- **Memory:** +100KB (negligible)
- **CPU (idle):** 0%
- **CPU (during sync):** <5% for ~2-5 seconds every 15min
- **Network:** Google Sheets API call (free tier: 100 req/100s)

### Total Additional Cost: **$0**

✅ **Recommendation:** In-process goroutine with time.Ticker

---

**Document Version:** 1.1
**Created:** 2025-02-15
**Updated:** 2025-02-15 (pickup time, statuses, cron solution)
**Author:** Claude (via warrmag)
**Status:** Refined — Ready for Phase 1 Implementation
