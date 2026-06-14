package equipment_requests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"warehouse/internal/auditlog"
	"warehouse/internal/inventory/assets"
	"warehouse/internal/inventory/category"
	inventorylog "warehouse/internal/inventory/inventory_log"
	"warehouse/internal/inventory/stocks"
	"warehouse/internal/inventory/transfers"
	"warehouse/internal/repository"
	"warehouse/internal/settings"
	"warehouse/internal/users"
)

// ─────────────────────────────────────────────
// Fake SheetReader — injects static rows
// ─────────────────────────────────────────────

type fakeSheetReader struct {
	rows [][]string
}

func (f *fakeSheetReader) FetchSheet(_, _ string) ([][]string, error) {
	return f.rows, nil
}

// sheetHeader returns the canonical Polish header row used by ColumnMapper.
func sheetHeader() []string {
	return []string{
		"Rzeczy", "Ilość", "Pawilon", "Miejsce", "Stan",
		"Godzina odbioru", "Dostawa do", "Osoba odpowiedzialna za budżet",
		"Do kogo ma trafić", "UWAGI",
	}
}

// sheetRow builds a data row matching the header order.
func sheetRow(item, qty, pavilion, location, status, pickup, delivery, budget, recipient, notes string) []string {
	return []string{item, qty, pavilion, location, status, pickup, delivery, budget, recipient, notes}
}

// ─────────────────────────────────────────────
// DB helpers
// ─────────────────────────────────────────────

func eqTestDBURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://postgres:pyrpyr@localhost:15432/pyrhouse_test?sslmode=disable"
}

func setupEQTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("postgres", eqTestDBURL())
	if err != nil {
		t.Skipf("Test database not available: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Skipf("Cannot connect to test database: %v", err)
	}
	cleanup := func() {
		_, _ = db.Exec("DELETE FROM equipment_request_items WHERE quest_id IN (SELECT id FROM equipment_request_quests WHERE destination_pavilion LIKE '__TEST__%')")
		_, _ = db.Exec("DELETE FROM equipment_request_category_mapping WHERE form_item_name LIKE '__TEST__%'")
		_, _ = db.Exec("DELETE FROM equipment_request_location_mapping WHERE pavilion LIKE '__TEST__%'")

		// transfers linked to test quests (via the quest_transfers join table)
		linkedTransfers := `transfer_id IN (
			SELECT qt.transfer_id FROM quest_transfers qt
			JOIN equipment_request_quests q ON q.quest_id = qt.quest_id
			WHERE q.destination_pavilion LIKE '__TEST__%')`
		_, _ = db.Exec("DELETE FROM transfer_users WHERE " + linkedTransfers)
		_, _ = db.Exec("DELETE FROM serialized_transfers WHERE " + linkedTransfers)
		_, _ = db.Exec("DELETE FROM non_serialized_transfers WHERE " + linkedTransfers)
		_, _ = db.Exec("DELETE FROM transfers WHERE from_location_id IN (SELECT id FROM locations WHERE name LIKE '__TEST__%')")

		_, _ = db.Exec("DELETE FROM equipment_request_quests WHERE destination_pavilion LIKE '__TEST__%'")

		_, _ = db.Exec("DELETE FROM non_serialized_items WHERE location_id IN (SELECT id FROM locations WHERE name LIKE '__TEST__%')")
		_, _ = db.Exec("DELETE FROM items WHERE item_serial LIKE '__TEST__%'")
		_, _ = db.Exec("DELETE FROM item_category WHERE label LIKE '__TEST__%'")
		_, _ = db.Exec("DELETE FROM locations WHERE name LIKE '__TEST__%'")
		_, _ = db.Exec("DELETE FROM users WHERE username LIKE '__test__%'")
		_ = db.Close()
	}
	return db, cleanup
}

// ─────────────────────────────────────────────
// Fixture types
// ─────────────────────────────────────────────

type eqFixtures struct {
	fromLocID  int
	toLocID    int
	categoryID int
	stockID    int
}

func createEQFixtures(t *testing.T, db *sql.DB) eqFixtures {
	t.Helper()

	var fromLocID, toLocID int
	require.NoError(t, db.QueryRow("INSERT INTO locations (name) VALUES ('__TEST__EQFrom') RETURNING id").Scan(&fromLocID))
	require.NoError(t, db.QueryRow("INSERT INTO locations (name) VALUES ('__TEST__EQTo') RETURNING id").Scan(&toLocID))

	var categoryID int
	require.NoError(t, db.QueryRow(
		"INSERT INTO item_category (item_category, label, pyr_id, category_type) VALUES ('__test__eq_cat', '__TEST__EQCat', 'EQ99', 'asset') ON CONFLICT (item_category) DO UPDATE SET label = EXCLUDED.label RETURNING id",
	).Scan(&categoryID))

	var originID int
	require.NoError(t, db.QueryRow("SELECT id FROM origins LIMIT 1").Scan(&originID))

	var stockID int
	require.NoError(t, db.QueryRow(
		"INSERT INTO non_serialized_items (item_category_id, location_id, quantity, origin_id, origin_suffix) VALUES ($1, $2, 50, $3, 'test') RETURNING id",
		categoryID, fromLocID, originID,
	).Scan(&stockID))

	return eqFixtures{fromLocID: fromLocID, toLocID: toLocID, categoryID: categoryID, stockID: stockID}
}

// ─────────────────────────────────────────────
// Service & router builders
// ─────────────────────────────────────────────

func newEQService(db *sql.DB, reader SheetReader) *Service {
	repo := repository.NewRepository(db)
	questRepo := NewRepository(repo)
	categoryRepo := category.NewCategoryRepository(repo)
	settingsRepo := settings.NewRepository(repo)

	svc := NewService(reader, categoryRepo, questRepo, settingsRepo, "fake-sheet-id", "Zamówienia", 3)
	return svc
}

func newEQServiceWithTransfers(db *sql.DB, reader SheetReader) *Service {
	repo := repository.NewRepository(db)

	transferRepo := transfers.NewRepository(repo)
	assetRepo := assets.NewRepository(repo)
	userRepo := users.NewRepository(repo)
	al := auditlog.NewAuditLog(auditlog.NewRepository(repo))
	il := inventorylog.NewInventoryLog(al)
	stockRepo := stocks.NewRepository(repo)

	transferSvc := transfers.NewService(repo, transferRepo, assetRepo, stockRepo, userRepo, il)

	svc := newEQService(db, reader)
	svc.SetTransferCreator(transferSvc)
	return svc
}

func newEQRouter(db *sql.DB, reader SheetReader) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("role", "admin"); c.Next() })
	h := NewHandler(newEQServiceWithTransfers(db, reader))
	h.RegisterRoutes(r.Group("/"))
	return r
}

func eqJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

func insertTestQuest(t *testing.T, db *sql.DB, locationID *int, status string) string {
	t.Helper()
	nano := time.Now().UnixNano()
	key := fmt.Sprintf("__TEST__pav|__TEST__loc|testrecipient|2099-01-01|%d", nano)
	questID := fmt.Sprintf("quest-%016x", nano)

	var locArg interface{} = nil
	if locationID != nil {
		locArg = *locationID
	}

	var id int
	require.NoError(t, db.QueryRow(`
		INSERT INTO equipment_request_quests
			(quest_key, quest_id, destination_pavilion, destination_location, recipient, delivery_date, status, location_id, location_resolved)
		VALUES ($1, $2, '__TEST__pav', '__TEST__loc', 'Test Recipient', '2099-01-01', $3, $4, $5)
		RETURNING id`,
		key, questID, status, locArg, locationID != nil,
	).Scan(&id))

	return questID
}

func insertTestQuestItem(t *testing.T, db *sql.DB, questDBID int, categoryID int, itemName string, qty int) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO equipment_request_items (quest_id, item_name, quantity, category_id, category_match_type)
		VALUES ($1, $2, $3, $4, 'exact')`,
		questDBID, itemName, qty, categoryID,
	)
	require.NoError(t, err)
}

func getQuestDBID(t *testing.T, db *sql.DB, questID string) int {
	t.Helper()
	var id int
	require.NoError(t, db.QueryRow("SELECT id FROM equipment_request_quests WHERE quest_id = $1", questID).Scan(&id))
	return id
}

// ─────────────────────────────────────────────
// TestSync_ParsesAndPersistsQuests
// ─────────────────────────────────────────────

func TestSync_ParsesAndPersistsQuests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupEQTestDB(t)
	defer cleanup()

	reader := &fakeSheetReader{rows: [][]string{
		sheetHeader(),
		// Quest 1 — two items, same destination/recipient/date
		sheetRow("__TEST__Laptop", "2", "__TEST__pav", "__TEST__loc", "Zamówione", "", "2099-01-01", "", "Test Recipient", ""),
		sheetRow("__TEST__Mouse", "3", "__TEST__pav", "__TEST__loc", "Zamówione", "", "2099-01-01", "", "Test Recipient", ""),
		// Quest 2 — different recipient
		sheetRow("__TEST__Projector", "1", "__TEST__pav", "__TEST__loc2", "Zamówione", "", "2099-01-01", "", "Other Recipient", ""),
	}}

	svc := newEQService(db, reader)
	result, err := svc.SyncQuestsToDatabase(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 2, result.Stats.Created)
	assert.Equal(t, 0, result.Stats.Updated)
	assert.Equal(t, 3, result.Stats.ItemsAdded)
	assert.Len(t, result.Quests, 2)

	// verify quests are in DB
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM equipment_request_quests WHERE destination_pavilion = '__TEST__pav'").Scan(&count))
	assert.Equal(t, 2, count)

	// verify items
	require.NoError(t, db.QueryRow(`
		SELECT COUNT(*) FROM equipment_request_items
		WHERE quest_id IN (SELECT id FROM equipment_request_quests WHERE destination_pavilion = '__TEST__pav')`).Scan(&count))
	assert.Equal(t, 3, count)

	// verify sync log was written
	var logCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM equipment_request_sync_log WHERE sheet_id = 'fake-sheet-id'").Scan(&logCount))
	assert.GreaterOrEqual(t, logCount, 1)
}

// ─────────────────────────────────────────────
// TestSync_Idempotent
// ─────────────────────────────────────────────

func TestSync_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupEQTestDB(t)
	defer cleanup()

	rows := [][]string{
		sheetHeader(),
		sheetRow("__TEST__Kabel", "5", "__TEST__pav", "__TEST__loc", "Zamówione", "", "2099-01-01", "", "Test Recipient", ""),
	}
	reader := &fakeSheetReader{rows: rows}
	svc := newEQService(db, reader)

	// first sync — creates
	r1, err := svc.SyncQuestsToDatabase(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, r1.Stats.Created)
	assert.Equal(t, 0, r1.Stats.Updated)

	// second sync — same data → unchanged
	r2, err := svc.SyncQuestsToDatabase(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, r2.Stats.Created)
	assert.Equal(t, 0, r2.Stats.Updated)
	assert.Equal(t, 1, r2.Stats.Unchanged)
}

// ─────────────────────────────────────────────
// TestSync_SkipsQuestWithLinkedTransfer
// ─────────────────────────────────────────────

func TestSync_SkipsQuestWithLinkedTransfer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupEQTestDB(t)
	defer cleanup()
	fx := createEQFixtures(t, db)

	// Insert a quest with a linked transfer directly (simulates in_progress state)
	rows := [][]string{
		sheetHeader(),
		sheetRow("__TEST__Kabel2", "3", "__TEST__pav", "__TEST__loc", "Zamówione", "", "2099-01-02", "", "Test Recipient", ""),
	}
	svc := newEQService(db, reader(&fakeSheetReader{rows: rows}))

	// first sync — creates quest
	r1, err := svc.SyncQuestsToDatabase(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, r1.Stats.Created)
	questID := r1.Quests[0].ID

	// manually link a fake transfer to the quest
	var dbQuestID int
	require.NoError(t, db.QueryRow("SELECT id FROM equipment_request_quests WHERE quest_id = $1", questID).Scan(&dbQuestID))

	var transferID int
	require.NoError(t, db.QueryRow(`INSERT INTO transfers (from_location_id, to_location_id, status) VALUES ($1, $2, 'in_transit') RETURNING id`,
		fx.fromLocID, fx.toLocID).Scan(&transferID))
	_, err = db.Exec("INSERT INTO quest_transfers (quest_id, transfer_id) VALUES ($1, $2)", questID, transferID)
	require.NoError(t, err)
	_, err = db.Exec("UPDATE equipment_request_quests SET status = 'in_progress' WHERE id = $1", dbQuestID)
	require.NoError(t, err)

	// second sync — same rows, but quest has transfer_id → should be skipped
	r2, err := svc.SyncQuestsToDatabase(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, r2.Stats.Created)
	assert.Equal(t, 0, r2.Stats.Updated)
	assert.Equal(t, 1, r2.Stats.Unchanged)

	// status must still be in_progress (not overwritten by sync)
	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM equipment_request_quests WHERE id = $1", dbQuestID).Scan(&status))
	assert.Equal(t, "in_progress", status)
}

// ─────────────────────────────────────────────
// TestSync_ReconcilesStalePendingDuplicate
// ─────────────────────────────────────────────

// Reproduces the production duplicate bug: a quest whose pickup time is edited in the
// sheet used to leave the pre-edit row behind forever. With pickup normalisation +
// reconciliation, the edited row replaces the old one instead of duplicating it.
func TestSync_ReconcilesStalePendingDuplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupEQTestDB(t)
	defer cleanup()
	_ = createEQFixtures(t, db)

	const recipient = "Dedup Recipient"

	// v1: pickup "10.00" (normalises to 10:00).
	svc1 := newEQService(db, reader(&fakeSheetReader{rows: [][]string{
		sheetHeader(),
		sheetRow("__TEST__Kabel", "2", "__TEST__pav", "__TEST__loc", "Nowe", "10.00", "2099-03-03", "", recipient, ""),
	}}))
	r1, err := svc1.SyncQuestsToDatabase(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, r1.Stats.Created)

	// v2: same recipient/location/date, pickup edited to "12:00:00" → different quest_key.
	svc2 := newEQService(db, reader(&fakeSheetReader{rows: [][]string{
		sheetHeader(),
		sheetRow("__TEST__Kabel", "2", "__TEST__pav", "__TEST__loc", "Nowe", "12:00:00", "2099-03-03", "", recipient, ""),
	}}))
	r2, err := svc2.SyncQuestsToDatabase(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, r2.Stats.Created, "edited pickup time creates a new quest")
	assert.Equal(t, 1, r2.Stats.Deleted, "the stale pre-edit quest is reconciled away")

	// Exactly one quest remains for this recipient — no duplicate.
	var count int
	require.NoError(t, db.QueryRow(
		"SELECT COUNT(*) FROM equipment_request_quests WHERE recipient = $1 AND destination_pavilion = '__TEST__pav'",
		recipient,
	).Scan(&count))
	assert.Equal(t, 1, count)

	// The survivor carries the normalised pickup time.
	var pickup string
	require.NoError(t, db.QueryRow(
		"SELECT pickup_time FROM equipment_request_quests WHERE recipient = $1 AND destination_pavilion = '__TEST__pav'",
		recipient,
	).Scan(&pickup))
	assert.Equal(t, "12:00", pickup)
}

// ─────────────────────────────────────────────
// TestSync_ReconcileKeepsQuestWithTransfer
// ─────────────────────────────────────────────

// Reconciliation must never remove a quest that has a linked transfer or a non-pending
// status, even when it no longer appears in the sheet.
func TestSync_ReconcileKeepsQuestWithTransfer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupEQTestDB(t)
	defer cleanup()
	fx := createEQFixtures(t, db)

	svc := newEQService(db, reader(&fakeSheetReader{rows: [][]string{
		sheetHeader(),
		sheetRow("__TEST__Kabel", "1", "__TEST__pav", "__TEST__loc", "Nowe", "", "2099-04-04", "", "Linked Recipient", ""),
	}}))
	r1, err := svc.SyncQuestsToDatabase(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, r1.Stats.Created)
	questID := r1.Quests[0].ID

	var transferID int
	require.NoError(t, db.QueryRow(`INSERT INTO transfers (from_location_id, to_location_id, status) VALUES ($1, $2, 'in_transit') RETURNING id`,
		fx.fromLocID, fx.toLocID).Scan(&transferID))
	_, err = db.Exec("INSERT INTO quest_transfers (quest_id, transfer_id) VALUES ($1, $2)", questID, transferID)
	require.NoError(t, err)
	_, err = db.Exec("UPDATE equipment_request_quests SET status = 'in_progress' WHERE quest_id = $1", questID)
	require.NoError(t, err)

	// Next sync with a completely different sheet — the linked quest is gone from it.
	svc2 := newEQService(db, reader(&fakeSheetReader{rows: [][]string{
		sheetHeader(),
		sheetRow("__TEST__Inny", "1", "__TEST__pav", "__TEST__loc", "Nowe", "", "2099-04-05", "", "Other Recipient", ""),
	}}))
	_, err = svc2.SyncQuestsToDatabase(context.Background())
	require.NoError(t, err)

	// The quest with a transfer must survive reconciliation.
	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM equipment_request_quests WHERE quest_id = $1", questID).Scan(&status))
	assert.Equal(t, "in_progress", status)
}

// ─────────────────────────────────────────────
// TestSync_KeepsItemWithoutQuantity
// ─────────────────────────────────────────────

// A sheet row with an item but a blank quantity must be imported (quantity NULL) instead of
// being dropped, and the transfer preview must flag it so the dispatcher fills it in.
func TestSync_KeepsItemWithoutQuantity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupEQTestDB(t)
	defer cleanup()
	fx := createEQFixtures(t, db)

	svc := newEQService(db, reader(&fakeSheetReader{rows: [][]string{
		sheetHeader(),
		sheetRow("__TEST__Kabel", "", "__TEST__pav", "__TEST__loc", "Nowe", "", "2099-05-05", "", "NoQty Recipient", ""),
	}}))
	r, err := svc.SyncQuestsToDatabase(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, r.Stats.Created)

	// Item is persisted with NULL quantity (not dropped).
	var rowCount int
	var qty sql.NullInt64
	require.NoError(t, db.QueryRow(`
		SELECT COUNT(*), MAX(i.quantity) FROM equipment_request_items i
		JOIN equipment_request_quests q ON q.id = i.quest_id
		WHERE q.destination_pavilion = '__TEST__pav' AND i.item_name = '__TEST__Kabel'`,
	).Scan(&rowCount, &qty))
	assert.Equal(t, 1, rowCount, "item with no quantity must still be imported")
	assert.False(t, qty.Valid, "quantity must be NULL when the sheet leaves it blank")

	// API model exposes the quantity as nil.
	require.Len(t, r.Quests, 1)
	require.Len(t, r.Quests[0].Items, 1)
	assert.Nil(t, r.Quests[0].Items[0].Quantity)

	// Transfer preview flags it so the dispatcher must supply a quantity.
	preview, err := svc.PreviewTransferFromQuest(context.Background(), r.Quests[0].ID, fx.fromLocID)
	require.NoError(t, err)
	require.Len(t, preview.UnresolvedItems, 1)
	assert.Equal(t, "quantity not specified in sheet", preview.UnresolvedItems[0].Reason)
	assert.Nil(t, preview.UnresolvedItems[0].Quantity)
}

// ─────────────────────────────────────────────
// TestSync_CategoryExactMatch
// ─────────────────────────────────────────────

func TestSync_CategoryExactMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupEQTestDB(t)
	defer cleanup()
	fx := createEQFixtures(t, db)

	// Item name exactly matches the category label "__TEST__EQCat"
	rows := [][]string{
		sheetHeader(),
		sheetRow("__TEST__EQCat", "2", "__TEST__pav", "__TEST__loc", "Zamówione", "", "2099-01-03", "", "Test Recipient", ""),
	}
	svc := newEQService(db, &fakeSheetReader{rows: rows})

	result, err := svc.SyncQuestsToDatabase(context.Background())
	require.NoError(t, err)
	require.Len(t, result.Quests, 1)

	// Verify category_id was resolved on the item
	var catID *int
	var matchType string
	require.NoError(t, db.QueryRow(`
		SELECT category_id, category_match_type
		FROM equipment_request_items
		WHERE quest_id IN (SELECT id FROM equipment_request_quests WHERE destination_pavilion = '__TEST__pav' AND delivery_date = '2099-01-03')
		LIMIT 1`).Scan(&catID, &matchType))

	require.NotNil(t, catID)
	assert.Equal(t, fx.categoryID, *catID)
	assert.Equal(t, "exact", matchType)
}

// helper avoids shadowing the local variable name in TestSync_SkipsQuestWithLinkedTransfer
func reader(r *fakeSheetReader) *fakeSheetReader { return r }

// ─────────────────────────────────────────────
// TestListQuests_WithFilters
// ─────────────────────────────────────────────

func TestListQuests_WithFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupEQTestDB(t)
	defer cleanup()
	fx := createEQFixtures(t, db)

	insertTestQuest(t, db, &fx.toLocID, "pending")
	insertTestQuest(t, db, &fx.toLocID, "completed")
	insertTestQuest(t, db, nil, "pending")

	router := newEQRouter(db, &fakeSheetReader{})

	t.Run("no filter returns all test quests", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/equipment-requests/quests", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.GreaterOrEqual(t, int(resp["count"].(float64)), 3)
	})

	t.Run("filter by status=pending", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/equipment-requests/quests?status=pending", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Quests []Quest `json:"quests"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		for _, q := range resp.Quests {
			assert.Equal(t, "pending", q.Status)
		}
	})

	t.Run("filter by location_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/equipment-requests/quests?location_id=%d", fx.toLocID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Quests []Quest `json:"quests"`
			Count  int     `json:"count"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.GreaterOrEqual(t, resp.Count, 2)
		for _, q := range resp.Quests {
			require.NotNil(t, q.LocationID)
			assert.Equal(t, fx.toLocID, *q.LocationID)
		}
	})

	t.Run("limit=1 returns one quest", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/equipment-requests/quests?limit=1", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(1), resp["limit"])
		assert.Len(t, resp["quests"], 1)
	})
}

// ─────────────────────────────────────────────
// TestUpdateQuestStatus
// ─────────────────────────────────────────────

func TestUpdateQuestStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupEQTestDB(t)
	defer cleanup()
	fx := createEQFixtures(t, db)
	router := newEQRouter(db, &fakeSheetReader{})

	t.Run("manual status change succeeds when no transfer linked", func(t *testing.T) {
		questID := insertTestQuest(t, db, &fx.toLocID, "pending")

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch,
			"/equipment-requests/quests/"+questID+"/status",
			eqJSON(t, map[string]any{"status": "cancelled"}))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var status string
		require.NoError(t, db.QueryRow("SELECT status FROM equipment_request_quests WHERE quest_id = $1", questID).Scan(&status))
		assert.Equal(t, "cancelled", status)
	})

	t.Run("status change blocked when transfer linked", func(t *testing.T) {
		questID := insertTestQuest(t, db, &fx.toLocID, "in_progress")
		dbQuestID := getQuestDBID(t, db, questID)

		var transferID int
		require.NoError(t, db.QueryRow(`INSERT INTO transfers (from_location_id, to_location_id, status) VALUES ($1, $2, 'in_transit') RETURNING id`,
			fx.fromLocID, fx.toLocID).Scan(&transferID))
		_, _ = db.Exec("INSERT INTO quest_transfers (quest_id, transfer_id) SELECT quest_id, $1 FROM equipment_request_quests WHERE id = $2", transferID, dbQuestID)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch,
			"/equipment-requests/quests/"+questID+"/status",
			eqJSON(t, map[string]any{"status": "cancelled"}))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("invalid status returns 400", func(t *testing.T) {
		questID := insertTestQuest(t, db, nil, "pending")

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch,
			"/equipment-requests/quests/"+questID+"/status",
			eqJSON(t, map[string]any{"status": "bogus"}))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// ─────────────────────────────────────────────
// TestUpdateQuestLocation
// ─────────────────────────────────────────────

func TestUpdateQuestLocation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupEQTestDB(t)
	defer cleanup()
	fx := createEQFixtures(t, db)
	router := newEQRouter(db, &fakeSheetReader{})

	t.Run("sets location and marks resolved", func(t *testing.T) {
		questID := insertTestQuest(t, db, nil, "pending")

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch,
			"/equipment-requests/quests/"+questID+"/location",
			eqJSON(t, map[string]any{"location_id": fx.toLocID, "save_mapping": false}))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var locID *int
		var resolved bool
		require.NoError(t, db.QueryRow("SELECT location_id, location_resolved FROM equipment_request_quests WHERE quest_id = $1", questID).Scan(&locID, &resolved))
		require.NotNil(t, locID)
		assert.Equal(t, fx.toLocID, *locID)
		assert.True(t, resolved)
	})

	t.Run("save_mapping=true creates location mapping", func(t *testing.T) {
		questID := insertTestQuest(t, db, nil, "pending")

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch,
			"/equipment-requests/quests/"+questID+"/location",
			eqJSON(t, map[string]any{"location_id": fx.toLocID, "save_mapping": true}))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var count int
		require.NoError(t, db.QueryRow(`
			SELECT COUNT(*) FROM equipment_request_location_mapping
			WHERE pavilion = '__TEST__pav' AND location_name = '__TEST__loc' AND location_id = $1`,
			fx.toLocID).Scan(&count))
		assert.GreaterOrEqual(t, count, 1)
	})
}

// ─────────────────────────────────────────────
// TestCategoryMapping_CRUD
// ─────────────────────────────────────────────

func TestCategoryMapping_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupEQTestDB(t)
	defer cleanup()
	fx := createEQFixtures(t, db)
	router := newEQRouter(db, &fakeSheetReader{})

	t.Run("create mapping", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/equipment-requests/category-mapping",
			eqJSON(t, map[string]any{"form_item_name": "__TEST__Przedłużacz", "category_id": fx.categoryID}))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("list mappings includes created", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/equipment-requests/category-mappings", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Mappings []CategoryMapping `json:"mappings"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		found := false
		for _, m := range resp.Mappings {
			if m.FormItemName == "__TEST__Przedłużacz" {
				found = true
				assert.Equal(t, fx.categoryID, m.CategoryID)
			}
		}
		assert.True(t, found)
	})

	t.Run("sync picks up manual mapping", func(t *testing.T) {
		// Row with exact item name matching the manual mapping
		reader := &fakeSheetReader{rows: [][]string{
			sheetHeader(),
			sheetRow("__TEST__Przedłużacz", "4", "__TEST__pav", "__TEST__map_loc", "Zamówione", "", "2099-01-10", "", "Test Recipient", ""),
		}}
		svc := newEQService(db, reader)
		result, err := svc.SyncQuestsToDatabase(context.Background())
		require.NoError(t, err)
		require.Len(t, result.Quests, 1)

		var catID *int
		var matchType string
		require.NoError(t, db.QueryRow(`
			SELECT category_id, category_match_type FROM equipment_request_items
			WHERE quest_id IN (SELECT id FROM equipment_request_quests WHERE delivery_date = '2099-01-10')
			LIMIT 1`).Scan(&catID, &matchType))

		require.NotNil(t, catID)
		assert.Equal(t, fx.categoryID, *catID)
		assert.Equal(t, "manual", matchType)
	})

	t.Run("delete mapping", func(t *testing.T) {
		// get the mapping id
		var mappingID int
		require.NoError(t, db.QueryRow("SELECT id FROM equipment_request_category_mapping WHERE form_item_name = '__TEST__Przedłużacz'").Scan(&mappingID))

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/equipment-requests/category-mappings/%d", mappingID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		var count int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM equipment_request_category_mapping WHERE id = $1", mappingID).Scan(&count))
		assert.Equal(t, 0, count)
	})
}

// ─────────────────────────────────────────────
// TestCreateTransferFromQuest_E2E
// ─────────────────────────────────────────────

func TestCreateTransferFromQuest_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupEQTestDB(t)
	defer cleanup()
	fx := createEQFixtures(t, db)
	router := newEQRouter(db, &fakeSheetReader{})

	t.Run("auto-resolves stock from quest items and creates transfer", func(t *testing.T) {
		questID := insertTestQuest(t, db, &fx.toLocID, "pending")
		dbQuestID := getQuestDBID(t, db, questID)
		insertTestQuestItem(t, db, dbQuestID, fx.categoryID, "__TEST__EQCat", 5)

		var beforeQty int
		require.NoError(t, db.QueryRow("SELECT quantity FROM non_serialized_items WHERE id = $1", fx.stockID).Scan(&beforeQty))

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost,
			"/equipment-requests/quests/"+questID+"/transfer",
			eqJSON(t, map[string]any{
				"from_location_id": fx.fromLocID,
				"to_location_id":   fx.toLocID,
			}))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code, "body: %s", w.Body.String())

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		transferID := int(resp["transfer_id"].(float64))
		assert.Greater(t, transferID, 0)

		// quest must now be linked and in_progress
		var status string
		var linkedTransferID *int
		require.NoError(t, db.QueryRow("SELECT q.status, qt.transfer_id FROM equipment_request_quests q LEFT JOIN quest_transfers qt ON qt.quest_id = q.quest_id WHERE q.quest_id = $1", questID).Scan(&status, &linkedTransferID))
		assert.Equal(t, "in_progress", status)
		require.NotNil(t, linkedTransferID)
		assert.Equal(t, transferID, *linkedTransferID)

		// stock was decremented at source
		var afterQty int
		require.NoError(t, db.QueryRow("SELECT quantity FROM non_serialized_items WHERE id = $1", fx.stockID).Scan(&afterQty))
		assert.Equal(t, beforeQty-5, afterQty)

		// transfer record exists with correct locations
		var fromLoc, toLoc int
		var tStatus string
		require.NoError(t, db.QueryRow("SELECT from_location_id, to_location_id, status FROM transfers WHERE id = $1", transferID).Scan(&fromLoc, &toLoc, &tStatus))
		assert.Equal(t, fx.fromLocID, fromLoc)
		assert.Equal(t, fx.toLocID, toLoc)
		assert.Equal(t, "in_transit", tStatus)
	})

	t.Run("explicit stock_items override used instead of auto-resolve", func(t *testing.T) {
		questID := insertTestQuest(t, db, &fx.toLocID, "pending")
		dbQuestID := getQuestDBID(t, db, questID)
		insertTestQuestItem(t, db, dbQuestID, fx.categoryID, "__TEST__EQCat", 10)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost,
			"/equipment-requests/quests/"+questID+"/transfer",
			eqJSON(t, map[string]any{
				"from_location_id": fx.fromLocID,
				"to_location_id":   fx.toLocID,
				"stock_items":      []map[string]any{{"id": fx.stockID, "quantity": 3}},
			}))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code, "body: %s", w.Body.String())
	})

	// A quest can now have multiple transfers (see migration 000046), so an in_progress
	// quest is no longer blocked. Only a completed/cancelled quest rejects new transfers.
	t.Run("completed quest returns 409", func(t *testing.T) {
		questID := insertTestQuest(t, db, &fx.toLocID, "completed")

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost,
			"/equipment-requests/quests/"+questID+"/transfer",
			eqJSON(t, map[string]any{"from_location_id": fx.fromLocID, "to_location_id": fx.toLocID}))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code, "body: %s", w.Body.String())
	})

	t.Run("quest not found returns 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost,
			"/equipment-requests/quests/quest-deadbeef/transfer",
			eqJSON(t, map[string]any{"from_location_id": fx.fromLocID}))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// ─────────────────────────────────────────────
// TestTransferCallback_QuestLifecycle
// ─────────────────────────────────────────────

func TestTransferCallback_QuestLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupEQTestDB(t)
	defer cleanup()
	fx := createEQFixtures(t, db)

	svc := newEQServiceWithTransfers(db, &fakeSheetReader{})

	setupLinkedQuest := func(t *testing.T) (questID string, transferID int) {
		t.Helper()
		questID = insertTestQuest(t, db, &fx.toLocID, "in_progress")
		require.NoError(t, db.QueryRow(`INSERT INTO transfers (from_location_id, to_location_id, status) VALUES ($1, $2, 'in_transit') RETURNING id`,
			fx.fromLocID, fx.toLocID).Scan(&transferID))
		dbQuestID := getQuestDBID(t, db, questID)
		_, err := db.Exec("INSERT INTO quest_transfers (quest_id, transfer_id) SELECT quest_id, $1 FROM equipment_request_quests WHERE id = $2", transferID, dbQuestID)
		require.NoError(t, err)
		return
	}

	t.Run("completed transfer marks quest completed", func(t *testing.T) {
		questID, transferID := setupLinkedQuest(t)

		// The transfer service flips the row to completed before firing the callback.
		_, err := db.Exec("UPDATE transfers SET status = 'completed' WHERE id = $1", transferID)
		require.NoError(t, err)

		err = svc.OnTransferStatusChanged(transferID, "completed")
		require.NoError(t, err)

		var status string
		require.NoError(t, db.QueryRow("SELECT status FROM equipment_request_quests WHERE quest_id = $1", questID).Scan(&status))
		assert.Equal(t, "completed", status)
	})

	t.Run("cancelled transfer resets quest to pending and unlinks transfer", func(t *testing.T) {
		questID, transferID := setupLinkedQuest(t)

		err := svc.OnTransferStatusChanged(transferID, "cancelled")
		require.NoError(t, err)

		var status string
		var linkedID *int
		require.NoError(t, db.QueryRow("SELECT q.status, qt.transfer_id FROM equipment_request_quests q LEFT JOIN quest_transfers qt ON qt.quest_id = q.quest_id WHERE q.quest_id = $1", questID).Scan(&status, &linkedID))
		assert.Equal(t, "pending", status)
		assert.Nil(t, linkedID, "transfer_id must be NULL after cancel")
	})

	t.Run("unknown transfer id is a no-op", func(t *testing.T) {
		err := svc.OnTransferStatusChanged(999999999, "completed")
		assert.NoError(t, err)
	})
}

// ─────────────────────────────────────────────
// TestPreviewTransferFromQuest
// ─────────────────────────────────────────────

func TestPreviewTransferFromQuest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupEQTestDB(t)
	defer cleanup()
	fx := createEQFixtures(t, db)
	router := newEQRouter(db, &fakeSheetReader{})

	questID := insertTestQuest(t, db, &fx.toLocID, "pending")
	dbQuestID := getQuestDBID(t, db, questID)
	insertTestQuestItem(t, db, dbQuestID, fx.categoryID, "__TEST__EQCat", 7)

	t.Run("returns resolved and unresolved items", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet,
			fmt.Sprintf("/equipment-requests/quests/%s/transfer-preview?from_location_id=%d", questID, fx.fromLocID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
		var preview TransferPreview
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &preview))
		assert.Equal(t, fx.fromLocID, preview.FromLocationID)
		assert.Len(t, preview.ResolvedItems, 1)
		assert.Equal(t, 7, preview.ResolvedItems[0].Quantity)
	})

	t.Run("missing from_location_id returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet,
			"/equipment-requests/quests/"+questID+"/transfer-preview", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
