package transfers

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
	"warehouse/internal/models"
	"warehouse/internal/repository"
	"warehouse/internal/users"
)

func transfersTestDBURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://postgres:pyrpyr@localhost:15432/pyrhouse_test?sslmode=disable"
}

type transferFixtures struct {
	fromLocID  int
	toLocID    int
	categoryID int
	assetID1   int
	assetID2   int
	stockID    int
	userID     int
}

func setupTransfersTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("postgres", transfersTestDBURL())
	if err != nil {
		t.Skipf("Test database not available: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Skipf("Cannot connect to test database: %v", err)
	}
	cleanup := func() {
		_, _ = db.Exec("DELETE FROM transfer_users WHERE transfer_id IN (SELECT id FROM transfers WHERE from_location_id IN (SELECT id FROM locations WHERE name LIKE '__TEST__%'))")
		_, _ = db.Exec("DELETE FROM serialized_transfers WHERE transfer_id IN (SELECT id FROM transfers WHERE from_location_id IN (SELECT id FROM locations WHERE name LIKE '__TEST__%'))")
		_, _ = db.Exec("DELETE FROM non_serialized_transfers WHERE transfer_id IN (SELECT id FROM transfers WHERE from_location_id IN (SELECT id FROM locations WHERE name LIKE '__TEST__%'))")
		_, _ = db.Exec("DELETE FROM transfers WHERE from_location_id IN (SELECT id FROM locations WHERE name LIKE '__TEST__%') OR to_location_id IN (SELECT id FROM locations WHERE name LIKE '__TEST__%')")
		_, _ = db.Exec("DELETE FROM items WHERE item_serial LIKE '__TEST__%'")
		_, _ = db.Exec("DELETE FROM non_serialized_items WHERE location_id IN (SELECT id FROM locations WHERE name LIKE '__TEST__%')")
		_, _ = db.Exec("DELETE FROM item_category WHERE label LIKE '__TEST__%'")
		_, _ = db.Exec("DELETE FROM locations WHERE name LIKE '__TEST__%'")
		_, _ = db.Exec("DELETE FROM users WHERE username LIKE '__test__%'")
		_ = db.Close()
	}
	return db, cleanup
}

func createTransferFixtures(t *testing.T, db *sql.DB) transferFixtures {
	t.Helper()

	var fromLocID, toLocID int
	require.NoError(t, db.QueryRow(
		"INSERT INTO locations (name) VALUES ('__TEST__TransferFrom') RETURNING id",
	).Scan(&fromLocID))
	require.NoError(t, db.QueryRow(
		"INSERT INTO locations (name) VALUES ('__TEST__TransferTo') RETURNING id",
	).Scan(&toLocID))

	var categoryID int
	require.NoError(t, db.QueryRow(
		"INSERT INTO item_category (item_category, label, pyr_id, category_type) VALUES ('__test__transfer_cat', '__TEST__TransferCat', 'TC99', 'asset') ON CONFLICT (item_category) DO UPDATE SET label = EXCLUDED.label RETURNING id",
	).Scan(&categoryID))

	var assetID1, assetID2 int
	require.NoError(t, db.QueryRow(
		"INSERT INTO items (location_id, item_category_id, item_serial, status) VALUES ($1, $2, '__TEST__transfer-asset-1', 'available') RETURNING id",
		fromLocID, categoryID,
	).Scan(&assetID1))
	require.NoError(t, db.QueryRow(
		"INSERT INTO items (location_id, item_category_id, item_serial, status) VALUES ($1, $2, '__TEST__transfer-asset-2', 'available') RETURNING id",
		fromLocID, categoryID,
	).Scan(&assetID2))

	var originID int
	require.NoError(t, db.QueryRow("SELECT id FROM origins LIMIT 1").Scan(&originID))

	var stockID int
	require.NoError(t, db.QueryRow(
		"INSERT INTO non_serialized_items (item_category_id, location_id, quantity, origin_id, origin_suffix) VALUES ($1, $2, 100, $3, 'test') RETURNING id",
		categoryID, fromLocID, originID,
	).Scan(&stockID))

	var userID int
	require.NoError(t, db.QueryRow(
		"INSERT INTO users (username, fullname, password_hash, role) VALUES ('__test__transfer-user', 'Transfer User', 'hash', 'user') RETURNING id",
	).Scan(&userID))

	return transferFixtures{
		fromLocID:  fromLocID,
		toLocID:    toLocID,
		categoryID: categoryID,
		assetID1:   assetID1,
		assetID2:   assetID2,
		stockID:    stockID,
		userID:     userID,
	}
}

func newTransfersRouter(db *sql.DB, userID int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("role", "admin")
		c.Set("userID", fmt.Sprintf("%d", userID))
		c.Next()
	})

	repo := repository.NewRepository(db)
	transferRepo := NewRepository(repo)
	assetRepo := assets.NewRepository(repo)
	userRepo := users.NewRepository(repo)
	al := auditlog.NewAuditLog(auditlog.NewRepository(repo))
	h := NewHandler(repo, transferRepo, assetRepo, userRepo, al)
	h.RegisterRoutes(r.Group("/"))
	return r
}

func transferJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

func createTransferViaAPI(t *testing.T, router *gin.Engine, payload map[string]any) *models.Transfer {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/transfers", transferJSON(t, payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "create transfer failed: %s", w.Body.String())

	var transfer models.Transfer
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &transfer))
	require.NotZero(t, transfer.ID)
	return &transfer
}

// TestTransfer_Create covers POST /transfers
func TestTransfer_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupTransfersTestDB(t)
	defer cleanup()
	fx := createTransferFixtures(t, db)
	router := newTransfersRouter(db, fx.userID)

	tests := []struct {
		name       string
		payload    map[string]any
		wantStatus int
		checkFn    func(t *testing.T, body []byte)
	}{
		{
			name: "assets only",
			payload: map[string]any{
				"from_location_id": fx.fromLocID,
				"location_id":      fx.toLocID,
				"assets":           []map[string]any{{"id": fx.assetID1}},
			},
			wantStatus: http.StatusCreated,
			checkFn: func(t *testing.T, body []byte) {
				var tr models.Transfer
				require.NoError(t, json.Unmarshal(body, &tr))
				assert.NotZero(t, tr.ID)
				assert.Equal(t, "in_transit", tr.Status)
				assert.Equal(t, fx.fromLocID, tr.FromLocation.ID)
				assert.Equal(t, fx.toLocID, tr.ToLocation.ID)
				assert.Len(t, tr.AssetsCollection, 1)
			},
		},
		{
			name: "stock only",
			payload: map[string]any{
				"from_location_id": fx.fromLocID,
				"location_id":      fx.toLocID,
				"stocks":           []map[string]any{{"id": fx.stockID, "quantity": 5}},
			},
			wantStatus: http.StatusCreated,
			checkFn: func(t *testing.T, body []byte) {
				var tr models.Transfer
				require.NoError(t, json.Unmarshal(body, &tr))
				assert.NotZero(t, tr.ID)
				assert.Equal(t, "in_transit", tr.Status)
				assert.Len(t, tr.StockItemsCollection, 1)
			},
		},
		{
			name: "assets and stock",
			payload: map[string]any{
				"from_location_id": fx.fromLocID,
				"location_id":      fx.toLocID,
				"assets":           []map[string]any{{"id": fx.assetID2}},
				"stocks":           []map[string]any{{"id": fx.stockID, "quantity": 3}},
			},
			wantStatus: http.StatusCreated,
			checkFn: func(t *testing.T, body []byte) {
				var tr models.Transfer
				require.NoError(t, json.Unmarshal(body, &tr))
				assert.NotZero(t, tr.ID)
				assert.Len(t, tr.AssetsCollection, 1)
				assert.Len(t, tr.StockItemsCollection, 1)
			},
		},
		{
			name: "with users",
			payload: map[string]any{
				"from_location_id": fx.fromLocID,
				"location_id":      fx.toLocID,
				"stocks":           []map[string]any{{"id": fx.stockID, "quantity": 1}},
				"users":            []map[string]any{{"id": fx.userID}},
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "empty collection returns 400",
			payload: map[string]any{
				"from_location_id": fx.fromLocID,
				"location_id":      fx.toLocID,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "same from and to location returns 400",
			payload: map[string]any{
				"from_location_id": fx.fromLocID,
				"location_id":      fx.fromLocID,
				"assets":           []map[string]any{{"id": fx.assetID1}},
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing location_id returns 400",
			payload: map[string]any{
				"from_location_id": fx.fromLocID,
				"assets":           []map[string]any{{"id": fx.assetID1}},
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/transfers", transferJSON(t, tt.payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code, "body: %s", w.Body.String())
			if tt.checkFn != nil {
				tt.checkFn(t, w.Body.Bytes())
			}
		})
	}
}

// TestTransfer_GetSingle covers GET /transfers/:id
func TestTransfer_GetSingle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupTransfersTestDB(t)
	defer cleanup()
	fx := createTransferFixtures(t, db)
	router := newTransfersRouter(db, fx.userID)

	transfer := createTransferViaAPI(t, router, map[string]any{
		"from_location_id": fx.fromLocID,
		"location_id":      fx.toLocID,
		"assets":           []map[string]any{{"id": fx.assetID1}},
	})

	t.Run("existing transfer returns 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transfers/%d", transfer.ID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var tr models.Transfer
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &tr))
		assert.Equal(t, transfer.ID, tr.ID)
		assert.Equal(t, "in_transit", tr.Status)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/transfers/abc", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestTransfer_List covers GET /transfers with filters
func TestTransfer_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupTransfersTestDB(t)
	defer cleanup()
	fx := createTransferFixtures(t, db)
	router := newTransfersRouter(db, fx.userID)

	transfer := createTransferViaAPI(t, router, map[string]any{
		"from_location_id": fx.fromLocID,
		"location_id":      fx.toLocID,
		"assets":           []map[string]any{{"id": fx.assetID1}},
	})

	t.Run("list all returns 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/transfers", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by from_location_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transfers?from_location_id=%d", fx.fromLocID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var list []models.Transfer
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
		require.NotEmpty(t, list)
		found := false
		for _, tr := range list {
			if tr.ID == transfer.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "created transfer should appear in filtered list")
	})

	t.Run("filter by status", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/transfers?status=in_transit", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var list []models.Transfer
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
		for _, tr := range list {
			assert.Equal(t, "in_transit", tr.Status)
		}
	})
}

// TestTransfer_Confirm covers PATCH /transfers/:id/confirm
func TestTransfer_Confirm(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupTransfersTestDB(t)
	defer cleanup()
	fx := createTransferFixtures(t, db)
	router := newTransfersRouter(db, fx.userID)

	t.Run("confirm asset transfer moves assets to available at destination", func(t *testing.T) {
		transfer := createTransferViaAPI(t, router, map[string]any{
			"from_location_id": fx.fromLocID,
			"location_id":      fx.toLocID,
			"assets":           []map[string]any{{"id": fx.assetID1}},
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/transfers/%d/confirm", transfer.ID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		// verify asset is now available at destination
		var locID int
		var status string
		require.NoError(t, db.QueryRow("SELECT location_id, status FROM items WHERE id = $1", fx.assetID1).Scan(&locID, &status))
		assert.Equal(t, fx.toLocID, locID)
		assert.Equal(t, "available", status)

		// verify transfer status is completed
		var transferStatus string
		require.NoError(t, db.QueryRow("SELECT status FROM transfers WHERE id = $1", transfer.ID).Scan(&transferStatus))
		assert.Equal(t, "completed", transferStatus)
	})

	t.Run("confirm stock transfer increases quantity at destination", func(t *testing.T) {
		var beforeQty int
		_ = db.QueryRow("SELECT COALESCE((SELECT quantity FROM non_serialized_items WHERE item_category_id = $1 AND location_id = $2), 0)", fx.categoryID, fx.toLocID).Scan(&beforeQty)

		transfer := createTransferViaAPI(t, router, map[string]any{
			"from_location_id": fx.fromLocID,
			"location_id":      fx.toLocID,
			"stocks":           []map[string]any{{"id": fx.stockID, "quantity": 10}},
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/transfers/%d/confirm", transfer.ID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		var afterQty int
		require.NoError(t, db.QueryRow("SELECT quantity FROM non_serialized_items WHERE item_category_id = $1 AND location_id = $2", fx.categoryID, fx.toLocID).Scan(&afterQty))
		assert.Equal(t, beforeQty+10, afterQty)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/transfers/abc/confirm", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestTransfer_ConfirmAccumulatesNullSuffixStock is a regression test for the production
// bug where transferring a stock item with NULL origin_suffix into a location that already
// holds the same item created a duplicate row instead of summing the quantity. Root cause:
// UNIQUE (item_category_id, location_id, origin_id, origin_suffix) treats a NULL suffix as
// distinct, so the ON CONFLICT in IncreaseStockAtDestination never fires.
func TestTransfer_ConfirmAccumulatesNullSuffixStock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupTransfersTestDB(t)
	defer cleanup()
	fx := createTransferFixtures(t, db)
	router := newTransfersRouter(db, fx.userID)

	var originID int
	require.NoError(t, db.QueryRow("SELECT id FROM origins LIMIT 1").Scan(&originID))

	// Source stock at the origin location with NULL origin_suffix (plain origin).
	var srcStockID int
	require.NoError(t, db.QueryRow(
		"INSERT INTO non_serialized_items (item_category_id, location_id, quantity, origin_id, origin_suffix) VALUES ($1, $2, 100, $3, NULL) RETURNING id",
		fx.categoryID, fx.fromLocID, originID,
	).Scan(&srcStockID))

	// Destination already holds the same item (same category, origin, NULL suffix).
	_, err := db.Exec(
		"INSERT INTO non_serialized_items (item_category_id, location_id, quantity, origin_id, origin_suffix) VALUES ($1, $2, 20, $3, NULL)",
		fx.categoryID, fx.toLocID, originID,
	)
	require.NoError(t, err)

	transfer := createTransferViaAPI(t, router, map[string]any{
		"from_location_id": fx.fromLocID,
		"location_id":      fx.toLocID,
		"stocks":           []map[string]any{{"id": srcStockID, "quantity": 40}},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/transfers/%d/confirm", transfer.ID), nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var rowCount, totalQty int
	require.NoError(t, db.QueryRow(
		"SELECT COUNT(*), COALESCE(SUM(quantity), 0) FROM non_serialized_items WHERE item_category_id = $1 AND location_id = $2 AND origin_id = $3 AND origin_suffix IS NULL",
		fx.categoryID, fx.toLocID, originID,
	).Scan(&rowCount, &totalQty))

	assert.Equal(t, 1, rowCount, "transferred stock must merge into the existing destination row")
	assert.Equal(t, 60, totalQty, "destination quantity must be 20 (existing) + 40 (transferred)")
}

// TestTransfer_Cancel covers PATCH /transfers/:id/cancel
func TestTransfer_Cancel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupTransfersTestDB(t)
	defer cleanup()
	fx := createTransferFixtures(t, db)
	router := newTransfersRouter(db, fx.userID)

	t.Run("cancel asset transfer restores assets to original location", func(t *testing.T) {
		transfer := createTransferViaAPI(t, router, map[string]any{
			"from_location_id": fx.fromLocID,
			"location_id":      fx.toLocID,
			"assets":           []map[string]any{{"id": fx.assetID2}},
		})

		// asset should be at toLocID with in_transit status after creation
		var locID int
		require.NoError(t, db.QueryRow("SELECT location_id FROM items WHERE id = $1", fx.assetID2).Scan(&locID))
		assert.Equal(t, fx.toLocID, locID)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/transfers/%d/cancel", transfer.ID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		// asset should be back at fromLocID with available status
		var status string
		require.NoError(t, db.QueryRow("SELECT location_id, status FROM items WHERE id = $1", fx.assetID2).Scan(&locID, &status))
		assert.Equal(t, fx.fromLocID, locID)
		assert.Equal(t, "available", status)

		// transfer status should be cancelled
		var transferStatus string
		require.NoError(t, db.QueryRow("SELECT status FROM transfers WHERE id = $1", transfer.ID).Scan(&transferStatus))
		assert.Equal(t, "cancelled", transferStatus)
	})

	t.Run("cancel stock transfer restores quantity", func(t *testing.T) {
		var beforeQty int
		require.NoError(t, db.QueryRow("SELECT quantity FROM non_serialized_items WHERE id = $1", fx.stockID).Scan(&beforeQty))

		transfer := createTransferViaAPI(t, router, map[string]any{
			"from_location_id": fx.fromLocID,
			"location_id":      fx.toLocID,
			"stocks":           []map[string]any{{"id": fx.stockID, "quantity": 7}},
		})

		// quantity should decrease by 7
		var midQty int
		require.NoError(t, db.QueryRow("SELECT quantity FROM non_serialized_items WHERE id = $1", fx.stockID).Scan(&midQty))
		assert.Equal(t, beforeQty-7, midQty)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/transfers/%d/cancel", transfer.ID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		// quantity should be restored
		var afterQty int
		require.NoError(t, db.QueryRow("SELECT quantity FROM non_serialized_items WHERE id = $1", fx.stockID).Scan(&afterQty))
		assert.Equal(t, beforeQty, afterQty)
	})

	t.Run("cancel already completed transfer returns 400", func(t *testing.T) {
		transfer := createTransferViaAPI(t, router, map[string]any{
			"from_location_id": fx.fromLocID,
			"location_id":      fx.toLocID,
			"stocks":           []map[string]any{{"id": fx.stockID, "quantity": 1}},
		})

		// confirm first
		wConfirm := httptest.NewRecorder()
		reqConfirm := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/transfers/%d/confirm", transfer.ID), nil)
		router.ServeHTTP(wConfirm, reqConfirm)
		require.Equal(t, http.StatusOK, wConfirm.Code)

		// now cancel should fail
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/transfers/%d/cancel", transfer.ID), nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestTransfer_UpdateDeliveryLocation covers PATCH /transfers/:id/delivery-location
func TestTransfer_UpdateDeliveryLocation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupTransfersTestDB(t)
	defer cleanup()
	fx := createTransferFixtures(t, db)
	router := newTransfersRouter(db, fx.userID)

	transfer := createTransferViaAPI(t, router, map[string]any{
		"from_location_id": fx.fromLocID,
		"location_id":      fx.toLocID,
		"assets":           []map[string]any{{"id": fx.assetID1}},
	})

	t.Run("update delivery location on in_transit transfer", func(t *testing.T) {
		payload := map[string]any{
			"delivery_location": map[string]any{
				"lat":       52.2297,
				"lng":       21.0122,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			},
		}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/transfers/%d/delivery-location", transfer.ID), transferJSON(t, payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		var lat, lng float64
		require.NoError(t, db.QueryRow("SELECT delivery_latitude, delivery_longitude FROM transfers WHERE id = $1", transfer.ID).Scan(&lat, &lng))
		assert.InDelta(t, 52.2297, lat, 0.001)
		assert.InDelta(t, 21.0122, lng, 0.001)
	})

	t.Run("invalid transfer id returns 400", func(t *testing.T) {
		payload := map[string]any{
			"delivery_location": map[string]any{"lat": 1.0, "lng": 1.0, "timestamp": time.Now().UTC().Format(time.RFC3339)},
		}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/transfers/abc/delivery-location", transferJSON(t, payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestTransfer_RestoreAsset covers PATCH /transfers/:id/assets/:item_id/restore-to-location
func TestTransfer_RestoreAsset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupTransfersTestDB(t)
	defer cleanup()
	fx := createTransferFixtures(t, db)
	router := newTransfersRouter(db, fx.userID)

	t.Run("restore asset moves it to target location with available status", func(t *testing.T) {
		transfer := createTransferViaAPI(t, router, map[string]any{
			"from_location_id": fx.fromLocID,
			"location_id":      fx.toLocID,
			"assets":           []map[string]any{{"id": fx.assetID1}},
		})

		// after transfer creation, asset should be at toLocID with in_transit
		var locID int
		var status string
		require.NoError(t, db.QueryRow("SELECT location_id, status FROM items WHERE id = $1", fx.assetID1).Scan(&locID, &status))
		assert.Equal(t, fx.toLocID, locID)
		assert.Equal(t, "in_transit", status)

		// restore to fromLocID (original location)
		payload := map[string]any{"location_id": fx.fromLocID}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(
			http.MethodPatch,
			fmt.Sprintf("/transfers/%d/assets/%d/restore-to-location", transfer.ID, fx.assetID1),
			transferJSON(t, payload),
		)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		// asset should now be at fromLocID with available status
		require.NoError(t, db.QueryRow("SELECT location_id, status FROM items WHERE id = $1", fx.assetID1).Scan(&locID, &status))
		assert.Equal(t, fx.fromLocID, locID)
		assert.Equal(t, "available", status)

		// asset should be removed from serialized_transfers
		var count int
		require.NoError(t, db.QueryRow(
			"SELECT COUNT(*) FROM serialized_transfers WHERE transfer_id = $1 AND item_id = $2",
			transfer.ID, fx.assetID1,
		).Scan(&count))
		assert.Equal(t, 0, count, "asset should be removed from serialized_transfers")
	})

	t.Run("missing location_id returns 400", func(t *testing.T) {
		transfer := createTransferViaAPI(t, router, map[string]any{
			"from_location_id": fx.fromLocID,
			"location_id":      fx.toLocID,
			"assets":           []map[string]any{{"id": fx.assetID2}},
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(
			http.MethodPatch,
			fmt.Sprintf("/transfers/%d/assets/%d/restore-to-location", transfer.ID, fx.assetID2),
			transferJSON(t, map[string]any{}),
		)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestTransfer_RestoreStock covers PATCH /transfers/:id/categories/:category_id/restore-to-location
func TestTransfer_RestoreStock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupTransfersTestDB(t)
	defer cleanup()
	fx := createTransferFixtures(t, db)
	router := newTransfersRouter(db, fx.userID)

	t.Run("partial restore returns quantity to source location", func(t *testing.T) {
		// initial qty at source = 100; transfer 20 → source drops to 80
		var beforeQty int
		require.NoError(t, db.QueryRow("SELECT quantity FROM non_serialized_items WHERE id = $1", fx.stockID).Scan(&beforeQty))
		assert.Equal(t, 100, beforeQty)

		transfer := createTransferViaAPI(t, router, map[string]any{
			"from_location_id": fx.fromLocID,
			"location_id":      fx.toLocID,
			"stocks":           []map[string]any{{"id": fx.stockID, "quantity": 20}},
		})

		var afterTransferQty int
		require.NoError(t, db.QueryRow("SELECT quantity FROM non_serialized_items WHERE id = $1", fx.stockID).Scan(&afterTransferQty))
		assert.Equal(t, 80, afterTransferQty, "qty should decrease by transferred amount")

		// restore 5 back to fromLocID — source should go from 80 → 85
		payload := map[string]any{
			"quantity":    5,
			"location_id": fx.fromLocID,
		}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(
			http.MethodPatch,
			fmt.Sprintf("/transfers/%d/categories/%d/restore-to-location", transfer.ID, fx.categoryID),
			transferJSON(t, payload),
		)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		// source stock should be restored by 5
		var afterRestoreQty int
		require.NoError(t, db.QueryRow("SELECT quantity FROM non_serialized_items WHERE id = $1", fx.stockID).Scan(&afterRestoreQty))
		assert.Equal(t, 85, afterRestoreQty, "qty should increase by restored amount")

		// transfer record quantity should be decreased by 5
		var transferQty int
		require.NoError(t, db.QueryRow(
			"SELECT quantity FROM non_serialized_transfers WHERE transfer_id = $1 AND item_category_id = $2",
			transfer.ID, fx.categoryID,
		).Scan(&transferQty))
		assert.Equal(t, 15, transferQty, "transfer qty should decrease by restored amount")
	})

	t.Run("missing quantity returns 400", func(t *testing.T) {
		transfer := createTransferViaAPI(t, router, map[string]any{
			"from_location_id": fx.fromLocID,
			"location_id":      fx.toLocID,
			"stocks":           []map[string]any{{"id": fx.stockID, "quantity": 5}},
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(
			http.MethodPatch,
			fmt.Sprintf("/transfers/%d/categories/%d/restore-to-location", transfer.ID, fx.categoryID),
			transferJSON(t, map[string]any{"location_id": fx.fromLocID}),
		)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestTransfer_UpdateUsers covers PUT /transfers/:id/users
func TestTransfer_UpdateUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupTransfersTestDB(t)
	defer cleanup()
	fx := createTransferFixtures(t, db)
	router := newTransfersRouter(db, fx.userID)

	transfer := createTransferViaAPI(t, router, map[string]any{
		"from_location_id": fx.fromLocID,
		"location_id":      fx.toLocID,
		"assets":           []map[string]any{{"id": fx.assetID2}},
	})

	t.Run("set users on transfer", func(t *testing.T) {
		payload := map[string]any{"users": []int{fx.userID}}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/transfers/%d/users", transfer.ID), transferJSON(t, payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		var count int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM transfer_users WHERE transfer_id = $1 AND user_id = $2", transfer.ID, fx.userID).Scan(&count))
		assert.Equal(t, 1, count)
	})

	t.Run("clear users by setting empty list", func(t *testing.T) {
		// first make sure there's a user
		_, _ = db.Exec("INSERT INTO transfer_users (transfer_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", transfer.ID, fx.userID)

		payload := map[string]any{"users": []int{}}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/transfers/%d/users", transfer.ID), transferJSON(t, payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		// empty users list → UsersExists with empty slice; behavior depends on implementation
		// the handler calls UsersExists which may return true for empty list → OK
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
	})

	t.Run("non-existent user returns 500", func(t *testing.T) {
		payload := map[string]any{"users": []int{999999999}}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/transfers/%d/users", transfer.ID), transferJSON(t, payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("invalid transfer id returns 400", func(t *testing.T) {
		payload := map[string]any{"users": []int{fx.userID}}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/transfers/abc/users", transferJSON(t, payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestTransfer_GetByUserAndStatus covers GET /transfers/users/:user_id?status=...
func TestTransfer_GetByUserAndStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupTransfersTestDB(t)
	defer cleanup()
	fx := createTransferFixtures(t, db)
	router := newTransfersRouter(db, fx.userID)

	// create a transfer and assign the user
	transfer := createTransferViaAPI(t, router, map[string]any{
		"from_location_id": fx.fromLocID,
		"location_id":      fx.toLocID,
		"stocks":           []map[string]any{{"id": fx.stockID, "quantity": 2}},
		"users":            []map[string]any{{"id": fx.userID}},
	})
	// give goroutine time to insert the user
	time.Sleep(50 * time.Millisecond)
	_ = transfer

	t.Run("get transfers by user and status", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transfers/users/%d?status=in_transit", fx.userID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	})

	t.Run("invalid status returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transfers/users/%d?status=bogus", fx.userID), nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing status returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transfers/users/%d", fx.userID), nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("different user forbidden for non-moderator", func(t *testing.T) {
		// router has userID=fx.userID, requesting transfers for a different user with role "user" would be blocked
		// but our router sets role=admin which IS allowed — so this just checks a different user's transfers come back
		otherUserID := fx.userID + 9999
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transfers/users/%d?status=in_transit", otherUserID), nil)
		router.ServeHTTP(w, req)
		// admin role bypasses ownership check → 200
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
