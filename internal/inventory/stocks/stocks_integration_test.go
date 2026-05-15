package stocks

import (
	"bytes"
	"database/sql"
	"encoding/json"
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

func stocksTestDBURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://postgres:pyrpyr@localhost:15432/pyrhouse_test?sslmode=disable"
}

type stocksFixtures struct {
	cableCategoryID int
	hdmiCategoryID  int
	plainOriginID   int
	plainOriginSlug string
	personalSlug    string
	locationID      int
}

func setupStocksTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("postgres", stocksTestDBURL())
	if err != nil {
		t.Skipf("Test database not available: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Skipf("Cannot connect to test database: %v", err)
	}
	cleanup := func() {
		_, _ = db.Exec("DELETE FROM non_serialized_items WHERE item_category_id IN (SELECT id FROM item_category WHERE label LIKE '__TEST__%')")
		_, _ = db.Exec("DELETE FROM item_category WHERE label LIKE '__TEST__%'")
		_, _ = db.Exec("DELETE FROM locations WHERE name LIKE '__TEST__%'")
		_, _ = db.Exec("DELETE FROM origins WHERE slug LIKE '__test__%'")
		_ = db.Close()
	}
	return db, cleanup
}

func createStocksFixtures(t *testing.T, db *sql.DB) stocksFixtures {
	t.Helper()

	var cableCategoryID int
	require.NoError(t, db.QueryRow(
		"INSERT INTO item_category (item_category, label, pyr_id, category_type) VALUES ('__test__cable', '__TEST__Cable', 'CAB', 'stock') ON CONFLICT (item_category) DO UPDATE SET label = EXCLUDED.label RETURNING id",
	).Scan(&cableCategoryID))

	var hdmiCategoryID int
	require.NoError(t, db.QueryRow(
		"INSERT INTO item_category (item_category, label, pyr_id, category_type) VALUES ('__test__hdmi', '__TEST__HDMI', 'HDM', 'stock') ON CONFLICT (item_category) DO UPDATE SET label = EXCLUDED.label RETURNING id",
	).Scan(&hdmiCategoryID))

	var plainOriginID int
	var plainOriginSlug string
	require.NoError(t, db.QueryRow(
		"SELECT id, slug FROM origins WHERE active = true AND allow_suffix = false LIMIT 1",
	).Scan(&plainOriginID, &plainOriginSlug))

	var personalSlug string
	err := db.QueryRow("SELECT slug FROM origins WHERE active = true AND allow_suffix = true LIMIT 1").Scan(&personalSlug)
	if err != nil {
		var personalID int
		require.NoError(t, db.QueryRow(
			"INSERT INTO origins (slug, label, allow_suffix, sort_order) VALUES ('__test__personal', 'Test Personal', true, 99) RETURNING id",
		).Scan(&personalID))
		personalSlug = "__test__personal"
	}

	// Use the default warehouse location (id=1 is typically the main stock location)
	locationID := 1

	return stocksFixtures{
		cableCategoryID: cableCategoryID,
		hdmiCategoryID:  hdmiCategoryID,
		plainOriginID:   plainOriginID,
		plainOriginSlug: plainOriginSlug,
		personalSlug:    personalSlug,
		locationID:      locationID,
	}
}

func newStocksRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("role", "admin")
		c.Next()
	})

	repo := repository.NewRepository(db)
	stockRepo := NewRepository(repo)
	al := auditlog.NewAuditLog(auditlog.NewRepository(repo))
	originsRepo := origins.NewRepository(repo)
	originsService := origins.NewService(originsRepo)
	ss := NewStockService(repo, stockRepo, al)
	h := NewStockHandler(repo, stockRepo, al, ss, originsService)
	h.RegisterRoutes(r.Group("/"))
	return r
}

func stockJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

// TestCreateStock covers POST /stocks with different categories and origins
func TestCreateStock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupStocksTestDB(t)
	defer cleanup()
	fx := createStocksFixtures(t, db)
	router := newStocksRouter(db)

	tests := []struct {
		name       string
		payload    map[string]any
		wantStatus int
		checkFn    func(t *testing.T, body []byte)
	}{
		{
			name: "cable with plain origin",
			payload: map[string]any{
				"category_id": fx.cableCategoryID,
				"quantity":    10,
				"origin":      fx.plainOriginSlug,
				"location_id": fx.locationID,
			},
			wantStatus: http.StatusCreated,
			checkFn: func(t *testing.T, body []byte) {
				var resp map[string]any
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.NotZero(t, resp["id"])
				assert.Equal(t, float64(10), resp["quantity"])
				assert.Equal(t, fx.plainOriginSlug, resp["origin"])

				var qty int
				require.NoError(t, db.QueryRow(
					"SELECT quantity FROM non_serialized_items WHERE item_category_id = $1 AND location_id = $2 AND origin_id = $3",
					fx.cableCategoryID, fx.locationID, fx.plainOriginID,
				).Scan(&qty))
				assert.Equal(t, 10, qty)
			},
		},
		{
			name: "hdmi with personal origin suffix",
			payload: map[string]any{
				"category_id": fx.hdmiCategoryID,
				"quantity":    5,
				"origin":      fx.personalSlug + "-tomek",
				"location_id": fx.locationID,
			},
			wantStatus: http.StatusCreated,
			checkFn: func(t *testing.T, body []byte) {
				var resp map[string]any
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.NotZero(t, resp["id"])
				assert.Equal(t, float64(5), resp["quantity"])
				assert.Equal(t, fx.personalSlug+"-tomek", resp["origin"])
			},
		},
		{
			name: "hdmi with same personal suffix — quantity accumulates (ON CONFLICT)",
			payload: map[string]any{
				"category_id": fx.hdmiCategoryID,
				"quantity":    3,
				"origin":      fx.personalSlug + "-tomek",
				"location_id": fx.locationID,
			},
			wantStatus: http.StatusCreated,
			checkFn: func(t *testing.T, body []byte) {
				var resp map[string]any
				require.NoError(t, json.Unmarshal(body, &resp))
				// ON CONFLICT (non-NULL origin_suffix) adds quantities: 5 + 3 = 8
				assert.Equal(t, float64(8), resp["quantity"])
			},
		},
		{
			name: "missing quantity returns 400",
			payload: map[string]any{
				"category_id": fx.cableCategoryID,
				"origin":      fx.plainOriginSlug,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing category_id returns 400",
			payload: map[string]any{
				"quantity": 5,
				"origin":   fx.plainOriginSlug,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid origin returns 400",
			payload: map[string]any{
				"category_id": fx.cableCategoryID,
				"quantity":    5,
				"origin":      "nonexistent-origin-xyz",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/stocks", stockJSON(t, tt.payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code, "body: %s", w.Body.String())
			if tt.checkFn != nil {
				tt.checkFn(t, w.Body.Bytes())
			}
		})
	}
}

// TestCreateStock_DifferentOriginsSameCategory verifies that cable with plain origin
// and cable with personal origin are stored as separate stock rows
func TestCreateStock_DifferentOriginsSameCategory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupStocksTestDB(t)
	defer cleanup()
	fx := createStocksFixtures(t, db)
	router := newStocksRouter(db)

	// Add cables from plain origin
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/stocks", stockJSON(t, map[string]any{
		"category_id": fx.cableCategoryID,
		"quantity":    20,
		"origin":      fx.plainOriginSlug,
		"location_id": fx.locationID,
	}))
	req1.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusCreated, w1.Code, "plain origin: %s", w1.Body.String())

	// Add cables from personal origin
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/stocks", stockJSON(t, map[string]any{
		"category_id": fx.cableCategoryID,
		"quantity":    7,
		"origin":      fx.personalSlug + "-kasia",
		"location_id": fx.locationID,
	}))
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusCreated, w2.Code, "personal origin: %s", w2.Body.String())

	// Two separate rows for the same category/location but different origins
	var rowCount int
	require.NoError(t, db.QueryRow(
		"SELECT COUNT(*) FROM non_serialized_items WHERE item_category_id = $1 AND location_id = $2",
		fx.cableCategoryID, fx.locationID,
	).Scan(&rowCount))
	assert.Equal(t, 2, rowCount, "expected separate stock rows for each origin")

	var plain, personal int
	require.NoError(t, db.QueryRow(
		"SELECT quantity FROM non_serialized_items WHERE item_category_id = $1 AND location_id = $2 AND origin_id = $3 AND origin_suffix IS NULL",
		fx.cableCategoryID, fx.locationID, fx.plainOriginID,
	).Scan(&plain))
	assert.Equal(t, 20, plain)

	require.NoError(t, db.QueryRow(
		"SELECT quantity FROM non_serialized_items WHERE item_category_id = $1 AND location_id = $2 AND origin_suffix = 'kasia'",
		fx.cableCategoryID, fx.locationID,
	).Scan(&personal))
	assert.Equal(t, 7, personal)
}
