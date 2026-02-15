# Equipment Request Integration — Quick Start

## Phase 1 Implementation Checklist

### Prerequisites
- ✅ Google Sheets API already integrated
- ✅ Service Account credentials configured

### Step 1: Create Module Structure
```bash
mkdir -p internal/equipment_requests
touch internal/equipment_requests/{models.go,service.go,handler.go,column_mapper.go}
```

### Step 2: Add Configuration
```env
# Add to .env
EQUIPMENT_REQUEST_SHEET_ID=16BytrbWmyWeBGnlSIDZn1Lnb5rdspoQu_rpc5m5Vtbc
EQUIPMENT_REQUEST_SHEET_NAME=Zamówienia
EQUIPMENT_REQUEST_SYNC_ENABLED=false  # manual trigger only in Phase 1
EQUIPMENT_REQUEST_FUZZY_THRESHOLD=3
```

### Step 3: Column Mapping (Hardcoded)
```go
// internal/equipment_requests/column_mapper.go
package equipment_requests

var ColumnMapping = map[string]string{
    "item":         "Rzeczy",
    "quantity":     "Ilość",
    "pavilion":     "Pawilon",
    "location":     "Miejsce",
    "status":       "Stan",
    "pickup_time":  "Godzina odbioru",
    "delivery_date": "Dostawa do",
    "budget_owner": "Osoba odpowiedzialna za budżet",
    "recipient":    "Do kogo ma trafić",
    "notes":        "UWAGI",
}
```

### Step 4: Implement Service
```go
// internal/equipment_requests/service.go
package equipment_requests

type Service struct {
    sheetReader *googlesheets.Service
    categoryRepo *category.CategoryRepository
}

func (s *Service) FetchAndAggregate(ctx context.Context) ([]Quest, error) {
    // 1. Fetch sheet
    // 2. Parse rows
    // 3. Match categories
    // 4. Aggregate into quests
    // 5. Return
}
```

### Step 5: Wire in DI Container
```go
// internal/di/container.go
equipmentRequestService := equipment_requests.NewService(
    container.GoogleSheetsService,
    container.CategoryRepository,
)
equipmentRequestHandler := equipment_requests.NewHandler(equipmentRequestService)
```

### Step 6: Add Routes
```go
// internal/routing/routes.go
protected.GET("/equipment-requests/sync", equipmentRequestHandler.ManualSync)
protected.GET("/equipment-requests/quests", equipmentRequestHandler.ListQuests)
```

### Step 7: Test Manually
```bash
# Trigger sync
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/equipment-requests/sync

# Get quests
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/equipment-requests/quests
```

---

## Phase 1 Deliverables

- [ ] Manual sync endpoint works
- [ ] Returns aggregated quests as JSON
- [ ] Category matching >70% accuracy
- [ ] Pickup time parsing handles flexible formats
- [ ] Budget owner shared per quest
- [ ] Status filtering works

**Time Estimate:** 1-2 days

---

## Moving to Phase 2

When ready:
1. Create DB migrations (see main plan)
2. Add Repository layer
3. Persist quests
4. Add transfer creation endpoint

**Time Estimate:** 2-3 days

---

## Moving to Phase 3 (Auto-sync)

Simplest implementation:

```go
// main.go
func main() {
    // ... existing setup ...
    
    if config.EquipmentRequestSyncEnabled {
        go startEquipmentRequestSync(container.EquipmentRequestService, config)
    }
    
    // ... start server ...
}

func startEquipmentRequestSync(service *equipment_requests.Service, cfg *config.Config) {
    interval, _ := time.ParseDuration(cfg.EquipmentRequestSyncInterval)
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for range ticker.C {
        ctx := context.Background()
        if err := service.SyncQuests(ctx); err != nil {
            log.Printf("[equipment-requests] Sync failed: %v", err)
        } else {
            log.Printf("[equipment-requests] Sync completed")
        }
    }
}
```

**Time Estimate:** 1 day

---

**Total Phase 1-3:** 4-6 days of development
