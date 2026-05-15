package assets

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"warehouse/internal/auditlog"
	"warehouse/internal/origins"
	"warehouse/internal/repository"
)

func assetsTestDBURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://postgres:pyrpyr@localhost:15432/pyrhouse_test?sslmode=disable"
}

type assetsFixtures struct {
	laptopCategoryID  int
	printerCategoryID int
	plainOriginSlug   string
	personalOriginID  int
	personalSlug      string
}

func setupAssetsTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("postgres", assetsTestDBURL())
	if err != nil {
		t.Skipf("Test database not available: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Skipf("Cannot connect to test database: %v", err)
	}
	cleanup := func() {
		_, _ = db.Exec("DELETE FROM items WHERE item_serial LIKE '__TEST__%'")
		_, _ = db.Exec("DELETE FROM items WHERE item_serial IS NULL AND item_category_id IN (SELECT id FROM item_category WHERE label LIKE '__TEST__%')")
		_, _ = db.Exec("DELETE FROM item_category WHERE label LIKE '__TEST__%'")
		_, _ = db.Exec("DELETE FROM origins WHERE slug LIKE '__test__%'")
		_ = db.Close()
	}
	return db, cleanup
}

func createAssetsFixtures(t *testing.T, db *sql.DB) assetsFixtures {
	t.Helper()

	var laptopCategoryID int
	require.NoError(t, db.QueryRow(
		"INSERT INTO item_category (item_category, label, pyr_id, category_type) VALUES ('__test__laptop', '__TEST__Laptop', 'LAP', 'asset') ON CONFLICT (item_category) DO UPDATE SET label = EXCLUDED.label RETURNING id",
	).Scan(&laptopCategoryID))

	var printerCategoryID int
	require.NoError(t, db.QueryRow(
		"INSERT INTO item_category (item_category, label, pyr_id, category_type) VALUES ('__test__printer', '__TEST__Printer', 'PRT', 'asset') ON CONFLICT (item_category) DO UPDATE SET label = EXCLUDED.label RETURNING id",
	).Scan(&printerCategoryID))

	// Use first available plain origin
	var plainOriginSlug string
	require.NoError(t, db.QueryRow("SELECT slug FROM origins WHERE active = true AND allow_suffix = false LIMIT 1").Scan(&plainOriginSlug))

	// Get or create a personal (allow_suffix) origin
	var personalOriginID int
	var personalSlug string
	err := db.QueryRow("SELECT id, slug FROM origins WHERE active = true AND allow_suffix = true LIMIT 1").Scan(&personalOriginID, &personalSlug)
	if err != nil {
		require.NoError(t, db.QueryRow(
			"INSERT INTO origins (slug, label, allow_suffix, sort_order) VALUES ('__test__personal', 'Test Personal', true, 99) RETURNING id",
		).Scan(&personalOriginID))
		personalSlug = "__test__personal"
	}

	return assetsFixtures{
		laptopCategoryID:  laptopCategoryID,
		printerCategoryID: printerCategoryID,
		plainOriginSlug:   plainOriginSlug,
		personalOriginID:  personalOriginID,
		personalSlug:      personalSlug,
	}
}

func newAssetsRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("role", "admin")
		c.Next()
	})

	repo := repository.NewRepository(db)
	assetRepo := NewRepository(repo)
	al := auditlog.NewAuditLog(auditlog.NewRepository(repo))
	originsRepo := origins.NewRepository(repo)
	originsService := origins.NewService(originsRepo)
	h := NewAssetHandler(repo, assetRepo, al, originsService)
	h.RegisterRoutes(r.Group("/"))
	return r
}

func assetJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

// TestCreateAsset_SingleAsset covers POST /assets for a single asset with different categories and origins
func TestCreateAsset_SingleAsset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupAssetsTestDB(t)
	defer cleanup()
	fx := createAssetsFixtures(t, db)
	router := newAssetsRouter(db)

	tests := []struct {
		name       string
		payload    map[string]any
		wantStatus int
		checkFn    func(t *testing.T, body []byte)
	}{
		{
			name: "laptop with plain origin",
			payload: map[string]any{
				"serial":      "__TEST__laptop-001",
				"category_id": fx.laptopCategoryID,
				"origin":      fx.plainOriginSlug,
			},
			wantStatus: http.StatusCreated,
			checkFn: func(t *testing.T, body []byte) {
				var resp map[string]any
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.NotZero(t, resp["id"])
				assert.Equal(t, "available", resp["status"])
				assert.Equal(t, fx.plainOriginSlug, resp["origin"])

				pyrCode, _ := resp["pyrcode"].(string)
				assert.Contains(t, pyrCode, "PYR-LAP")

				var count int
				require.NoError(t, db.QueryRow(
					"SELECT COUNT(*) FROM items WHERE item_serial = '__TEST__laptop-001' AND item_category_id = $1",
					fx.laptopCategoryID,
				).Scan(&count))
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "printer with personal origin suffix",
			payload: map[string]any{
				"serial":      "__TEST__printer-001",
				"category_id": fx.printerCategoryID,
				"origin":      fx.personalSlug + "-jan",
			},
			wantStatus: http.StatusCreated,
			checkFn: func(t *testing.T, body []byte) {
				var resp map[string]any
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.NotZero(t, resp["id"])
				assert.Equal(t, "available", resp["status"])
				assert.Equal(t, fx.personalSlug+"-jan", resp["origin"])

				pyrCode, _ := resp["pyrcode"].(string)
				assert.Contains(t, pyrCode, "PYR-PRT")
			},
		},
		{
			name: "missing serial returns 400",
			payload: map[string]any{
				"category_id": fx.laptopCategoryID,
				"origin":      fx.plainOriginSlug,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid origin returns 400",
			payload: map[string]any{
				"serial":      "__TEST__laptop-bad-origin",
				"category_id": fx.laptopCategoryID,
				"origin":      "nonexistent-origin-xyz",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "duplicate serial returns 409",
			payload: map[string]any{
				"serial":      "__TEST__laptop-001",
				"category_id": fx.laptopCategoryID,
				"origin":      fx.plainOriginSlug,
			},
			wantStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/assets", assetJSON(t, tt.payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code, "body: %s", w.Body.String())
			if tt.checkFn != nil {
				tt.checkFn(t, w.Body.Bytes())
			}
		})
	}
}

// TestCreateBulkAssets covers POST /assets/bulk for multiple assets at once
func TestCreateBulkAssets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupAssetsTestDB(t)
	defer cleanup()
	fx := createAssetsFixtures(t, db)
	router := newAssetsRouter(db)

	t.Run("two laptops with plain origin", func(t *testing.T) {
		payload := map[string]any{
			"serials":     []string{"__TEST__bulk-laptop-A", "__TEST__bulk-laptop-B"},
			"category_id": fx.laptopCategoryID,
			"origin":      fx.plainOriginSlug,
		}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/assets/bulk", assetJSON(t, payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code, "body: %s", w.Body.String())

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

		created, ok := resp["created"].([]any)
		require.True(t, ok)
		assert.Len(t, created, 2)

		// Both assets should be in DB
		var count int
		require.NoError(t, db.QueryRow(
			"SELECT COUNT(*) FROM items WHERE item_serial IN ('__TEST__bulk-laptop-A', '__TEST__bulk-laptop-B') AND item_category_id = $1",
			fx.laptopCategoryID,
		).Scan(&count))
		assert.Equal(t, 2, count)

		// Both should have PYR codes
		for _, item := range created {
			asset := item.(map[string]any)
			pyrCode, _ := asset["pyrcode"].(string)
			assert.Contains(t, pyrCode, "PYR-LAP", fmt.Sprintf("asset %v missing pyrcode", asset["id"]))
		}
	})

	t.Run("two printers with personal origin suffix", func(t *testing.T) {
		payload := map[string]any{
			"serials":     []string{"__TEST__bulk-printer-A", "__TEST__bulk-printer-B"},
			"category_id": fx.printerCategoryID,
			"origin":      fx.personalSlug + "-anna",
		}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/assets/bulk", assetJSON(t, payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code, "body: %s", w.Body.String())

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

		created, ok := resp["created"].([]any)
		require.True(t, ok)
		assert.Len(t, created, 2)

		for _, item := range created {
			asset := item.(map[string]any)
			assert.Equal(t, fx.personalSlug+"-anna", asset["origin"])
		}
	})

	t.Run("one duplicate in bulk aborts all — no assets created", func(t *testing.T) {
		// laptop-A already exists from the first subtest
		payload := map[string]any{
			"serials":     []string{"__TEST__bulk-laptop-A", "__TEST__bulk-laptop-NEW"},
			"category_id": fx.laptopCategoryID,
			"origin":      fx.plainOriginSlug,
		}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/assets/bulk", assetJSON(t, payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())

		// __TEST__bulk-laptop-NEW must NOT be in DB
		var count int
		require.NoError(t, db.QueryRow(
			"SELECT COUNT(*) FROM items WHERE item_serial = '__TEST__bulk-laptop-NEW'",
		).Scan(&count))
		assert.Equal(t, 0, count)
	})

	t.Run("stock category type returns 400", func(t *testing.T) {
		var stockCategoryID int
		err := db.QueryRow(
			"INSERT INTO item_category (item_category, label, pyr_id, category_type) VALUES ('__test__cable_bulk_check', '__TEST__CableBulkCheck', 'CBB', 'stock') ON CONFLICT (item_category) DO UPDATE SET label = EXCLUDED.label RETURNING id",
		).Scan(&stockCategoryID)
		require.NoError(t, err)

		payload := map[string]any{
			"serials":     []string{"__TEST__should-not-create"},
			"category_id": stockCategoryID,
			"origin":      fx.plainOriginSlug,
		}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/assets/bulk", assetJSON(t, payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestCreateAssetWithoutSerial_Integration covers POST /assets/without-serial
func TestCreateAssetWithoutSerial_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupAssetsTestDB(t)
	defer cleanup()
	fx := createAssetsFixtures(t, db)
	router := newAssetsRouter(db)

	t.Run("creates 1 laptop without serial", func(t *testing.T) {
		payload := map[string]any{
			"quantity":    1,
			"category_id": fx.laptopCategoryID,
			"origin":      fx.plainOriginSlug,
		}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/assets/without-serial", assetJSON(t, payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code, "body: %s", w.Body.String())

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

		created, ok := resp["created"].([]any)
		require.True(t, ok)
		require.Len(t, created, 1)

		asset := created[0].(map[string]any)
		assert.Nil(t, asset["serial"])
		pyrCode, _ := asset["pyrcode"].(string)
		assert.Contains(t, pyrCode, "PYR-LAP")

		var count int
		require.NoError(t, db.QueryRow(
			"SELECT COUNT(*) FROM items WHERE item_serial IS NULL AND item_category_id = $1 AND status = 'available'",
			fx.laptopCategoryID,
		).Scan(&count))
		assert.Equal(t, 1, count)
	})

	t.Run("creates 3 printers without serial", func(t *testing.T) {
		payload := map[string]any{
			"quantity":    3,
			"category_id": fx.printerCategoryID,
			"origin":      fx.personalSlug + "-jan",
		}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/assets/without-serial", assetJSON(t, payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code, "body: %s", w.Body.String())

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

		created, ok := resp["created"].([]any)
		require.True(t, ok)
		assert.Len(t, created, 3)

		for _, item := range created {
			asset := item.(map[string]any)
			pyrCode, _ := asset["pyrcode"].(string)
			assert.Contains(t, pyrCode, "PYR-PRT")
		}

		var count int
		require.NoError(t, db.QueryRow(
			"SELECT COUNT(*) FROM items WHERE item_serial IS NULL AND item_category_id = $1",
			fx.printerCategoryID,
		).Scan(&count))
		assert.Equal(t, 3, count)
	})

	t.Run("quantity=0 returns 400", func(t *testing.T) {
		payload := map[string]any{
			"quantity":    0,
			"category_id": fx.laptopCategoryID,
			"origin":      fx.plainOriginSlug,
		}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/assets/without-serial", assetJSON(t, payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("stock category returns 400", func(t *testing.T) {
		var stockCatID int
		require.NoError(t, db.QueryRow(
			"INSERT INTO item_category (item_category, label, pyr_id, category_type) VALUES ('__test__cable_ws', '__TEST__CableWS', 'CWS', 'stock') ON CONFLICT (item_category) DO UPDATE SET label = EXCLUDED.label RETURNING id",
		).Scan(&stockCatID))

		payload := map[string]any{
			"quantity":    1,
			"category_id": stockCatID,
			"origin":      fx.plainOriginSlug,
		}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/assets/without-serial", assetJSON(t, payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestUpdateAssetSerial_Integration covers PATCH /assets/:id/serial
func TestUpdateAssetSerial_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupAssetsTestDB(t)
	defer cleanup()
	fx := createAssetsFixtures(t, db)
	router := newAssetsRouter(db)

	// Helper: create an asset without serial via the API
	createWithoutSerial := func(t *testing.T, categoryID int) int {
		t.Helper()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/assets/without-serial", assetJSON(t, map[string]any{
			"quantity":    1,
			"category_id": categoryID,
			"origin":      fx.plainOriginSlug,
		}))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		created := resp["created"].([]any)
		asset := created[0].(map[string]any)
		return int(asset["id"].(float64))
	}

	t.Run("assigns serial to laptop without serial", func(t *testing.T) {
		assetID := createWithoutSerial(t, fx.laptopCategoryID)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/assets/%d/serial", assetID), assetJSON(t, map[string]any{
			"serial": "__TEST__serial-assigned-001",
		}))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		var serial string
		require.NoError(t, db.QueryRow("SELECT item_serial FROM items WHERE id = $1", assetID).Scan(&serial))
		assert.Equal(t, "__TEST__serial-assigned-001", serial)
	})

	t.Run("duplicate serial returns 409", func(t *testing.T) {
		// Create two assets without serial
		id1 := createWithoutSerial(t, fx.laptopCategoryID)
		id2 := createWithoutSerial(t, fx.laptopCategoryID)

		// Assign serial to first
		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/assets/%d/serial", id1), assetJSON(t, map[string]any{
			"serial": "__TEST__serial-dup-001",
		}))
		req1.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w1, req1)
		require.Equal(t, http.StatusOK, w1.Code)

		// Try to assign same serial to second
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/assets/%d/serial", id2), assetJSON(t, map[string]any{
			"serial": "__TEST__serial-dup-001",
		}))
		req2.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusConflict, w2.Code)
	})

	t.Run("missing serial body returns 400", func(t *testing.T) {
		assetID := createWithoutSerial(t, fx.printerCategoryID)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/assets/%d/serial", assetID), assetJSON(t, map[string]any{}))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
