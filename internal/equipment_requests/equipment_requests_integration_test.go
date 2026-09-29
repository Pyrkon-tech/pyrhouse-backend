package equipment_requests

import (
	"bytes"
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
	inventorylog "warehouse/internal/inventory/inventory_log"
	"warehouse/internal/inventory/stocks"
	"warehouse/internal/inventory/transfers"
	"warehouse/internal/repository"
	"warehouse/internal/users"
)

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

func newEQService(db *sql.DB) *Service {
	return NewService(NewRepository(repository.NewRepository(db)))
}

func newEQServiceWithTransfers(db *sql.DB) *Service {
	repo := repository.NewRepository(db)

	transferRepo := transfers.NewRepository(repo)
	assetRepo := assets.NewRepository(repo)
	userRepo := users.NewRepository(repo)
	al := auditlog.NewAuditLog(auditlog.NewRepository(repo))
	il := inventorylog.NewInventoryLog(al)
	stockRepo := stocks.NewRepository(repo)

	transferSvc := transfers.NewService(repo, transferRepo, assetRepo, stockRepo, userRepo, il)

	svc := newEQService(db)
	svc.SetTransferCreator(transferSvc)
	return svc
}

func newEQRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("role", "admin"); c.Next() })
	h := NewHandler(newEQServiceWithTransfers(db))
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
			(quest_key, quest_id, destination_pavilion, destination_location, recipient, delivery_date, status, location_id, location_resolved, source)
		VALUES ($1, $2, '__TEST__pav', '__TEST__loc', 'Test Recipient', '2099-01-01', $3, $4, $5, 'sheet')
		RETURNING id`,
		key, questID, status, locArg, locationID != nil,
	).Scan(&id))

	return questID
}

func insertTestQuestItem(t *testing.T, db *sql.DB, questDBID int, categoryID int, itemName string, qty int) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO equipment_request_items (quest_id, item_name, quantity, category_id)
		VALUES ($1, $2, $3, $4)`,
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

	router := newEQRouter(db)

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
	router := newEQRouter(db)

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
	router := newEQRouter(db)

	t.Run("sets location and marks resolved", func(t *testing.T) {
		questID := insertTestQuest(t, db, nil, "pending")

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch,
			"/equipment-requests/quests/"+questID+"/location",
			eqJSON(t, map[string]any{"location_id": fx.toLocID}))
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
	router := newEQRouter(db)

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

	svc := newEQServiceWithTransfers(db)

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
	router := newEQRouter(db)

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
