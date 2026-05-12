package origins

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
	"warehouse/internal/repository"
)

func originsTestDBURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://postgres:pyrpyr@localhost:15432/pyrhouse_test?sslmode=disable"
}

func setupOriginsTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("postgres", originsTestDBURL())
	if err != nil {
		t.Skipf("Test database not available: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Skipf("Cannot connect to test database: %v", err)
	}
	cleanup := func() {
		_, _ = db.Exec("DELETE FROM items WHERE item_serial LIKE '__TEST__%'")
		_, _ = db.Exec("DELETE FROM origins WHERE slug LIKE '__test__%'")
		_ = db.Close()
	}
	return db, cleanup
}

func newOriginsRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("role", "admin"); c.Next() })

	repo := repository.NewRepository(db)
	h := NewHandler(NewService(NewRepository(repo)))
	h.RegisterRoutes(r.Group("/"))
	return r
}

func origJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

func createTestOrigin(t *testing.T, db *sql.DB, slug string, sortOrder int) int {
	t.Helper()
	var id int
	err := db.QueryRow(
		"INSERT INTO origins (slug, label, sort_order) VALUES ($1, $2, $3) RETURNING id",
		slug, "Test "+slug, sortOrder,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestOrigins_Create covers POST /origins
func TestOrigins_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupOriginsTestDB(t)
	defer cleanup()
	router := newOriginsRouter(db)

	tests := []struct {
		name       string
		payload    map[string]any
		wantStatus int
		checkFn    func(t *testing.T, body []byte)
	}{
		{
			name:       "valid origin",
			payload:    map[string]any{"slug": "__test__alpha", "label": "Test Alpha", "sort_order": 999},
			wantStatus: http.StatusCreated,
			checkFn: func(t *testing.T, body []byte) {
				var o Origin
				require.NoError(t, json.Unmarshal(body, &o))
				assert.NotZero(t, o.ID)
				assert.Equal(t, "__test__alpha", o.Slug)
				assert.Equal(t, "Test Alpha", o.Label)
				assert.Equal(t, 999, o.SortOrder)
				assert.True(t, o.Active)
			},
		},
		{
			name:       "slug is lowercased and trimmed",
			payload:    map[string]any{"slug": "  __TEST__Beta  ", "label": "Test Beta"},
			wantStatus: http.StatusCreated,
			checkFn: func(t *testing.T, body []byte) {
				var o Origin
				require.NoError(t, json.Unmarshal(body, &o))
				assert.Equal(t, "__test__beta", o.Slug)
			},
		},
		{
			name:       "duplicate slug returns 409",
			payload:    map[string]any{"slug": "__test__alpha", "label": "Duplicate"},
			wantStatus: http.StatusConflict,
		},
		{
			name:       "missing slug returns 400",
			payload:    map[string]any{"label": "No Slug"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing label returns 400",
			payload:    map[string]any{"slug": "__test__nolabel"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/origins", origJSON(t, tt.payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.checkFn != nil {
				tt.checkFn(t, w.Body.Bytes())
			}
		})
	}
}

// TestOrigins_ListActive covers GET /origins — only active, sorted by sort_order
func TestOrigins_ListActive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupOriginsTestDB(t)
	defer cleanup()

	// insert with out-of-order sort_order values to verify ordering
	createTestOrigin(t, db, "__test__list-c", 1003)
	createTestOrigin(t, db, "__test__list-a", 1001)
	createTestOrigin(t, db, "__test__list-b", 1002)

	// insert an inactive origin — should NOT appear in ListActive
	_, err := db.Exec(
		"INSERT INTO origins (slug, label, sort_order, active) VALUES ('__test__inactive', 'Inactive', 1004, false)",
	)
	require.NoError(t, err)

	router := newOriginsRouter(db)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/origins", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var origins []Origin
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &origins))

	// collect test origins from response (filter by __test__ prefix)
	var testOrigins []Origin
	for _, o := range origins {
		if len(o.Slug) >= 8 && o.Slug[:8] == "__test__" {
			testOrigins = append(testOrigins, o)
		}
	}

	require.Len(t, testOrigins, 3, "expected 3 active test origins")

	// verify sort_order ascending
	assert.Equal(t, "__test__list-a", testOrigins[0].Slug)
	assert.Equal(t, "__test__list-b", testOrigins[1].Slug)
	assert.Equal(t, "__test__list-c", testOrigins[2].Slug)

	// verify inactive is absent
	for _, o := range origins {
		assert.NotEqual(t, "__test__inactive", o.Slug)
	}
}

// TestOrigins_ListAll covers GET /origins/all — includes inactive
func TestOrigins_ListAll(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupOriginsTestDB(t)
	defer cleanup()

	createTestOrigin(t, db, "__test__all-active", 1010)
	_, err := db.Exec(
		"INSERT INTO origins (slug, label, sort_order, active) VALUES ('__test__all-inactive', 'All Inactive', 1011, false)",
	)
	require.NoError(t, err)

	router := newOriginsRouter(db)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/origins/all", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var origins []Origin
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &origins))

	slugs := make([]string, len(origins))
	for i, o := range origins {
		slugs[i] = o.Slug
	}
	assert.Contains(t, slugs, "__test__all-active")
	assert.Contains(t, slugs, "__test__all-inactive")
}

// TestOrigins_Update covers PATCH /origins/:id
func TestOrigins_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupOriginsTestDB(t)
	defer cleanup()

	id := createTestOrigin(t, db, "__test__update-orig", 2000)
	router := newOriginsRouter(db)

	tests := []struct {
		name       string
		id         int
		payload    map[string]any
		wantStatus int
		checkFn    func(t *testing.T, body []byte)
	}{
		{
			name:       "update label",
			id:         id,
			payload:    map[string]any{"label": "Updated Label"},
			wantStatus: http.StatusOK,
			checkFn: func(t *testing.T, body []byte) {
				var o Origin
				require.NoError(t, json.Unmarshal(body, &o))
				assert.Equal(t, "Updated Label", o.Label)
			},
		},
		{
			name:       "update sort_order",
			id:         id,
			payload:    map[string]any{"sort_order": 2999},
			wantStatus: http.StatusOK,
			checkFn: func(t *testing.T, body []byte) {
				var o Origin
				require.NoError(t, json.Unmarshal(body, &o))
				assert.Equal(t, 2999, o.SortOrder)
			},
		},
		{
			name:       "deactivate",
			id:         id,
			payload:    map[string]any{"active": false},
			wantStatus: http.StatusOK,
			checkFn: func(t *testing.T, body []byte) {
				var o Origin
				require.NoError(t, json.Unmarshal(body, &o))
				assert.False(t, o.Active)
			},
		},
		{
			name:       "empty payload returns 400",
			id:         id,
			payload:    map[string]any{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "non-existent id returns 404",
			id:         999999999,
			payload:    map[string]any{"label": "Ghost"},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "non-numeric id returns 400",
			id:         -1, // will be replaced inline
			payload:    map[string]any{"label": "Bad"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("/origins/%d", tt.id)
			if tt.name == "non-numeric id returns 400" {
				path = "/origins/abc"
			}
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPatch, path, origJSON(t, tt.payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.checkFn != nil {
				tt.checkFn(t, w.Body.Bytes())
			}
		})
	}
}

// TestOrigins_Delete covers DELETE /origins/:id
func TestOrigins_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupOriginsTestDB(t)
	defer cleanup()
	router := newOriginsRouter(db)

	t.Run("empty origin is deleted", func(t *testing.T) {
		id := createTestOrigin(t, db, "__test__del-empty", 3000)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/origins/%d", id), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM origins WHERE id = $1", id).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("origin with assigned item is blocked with 409", func(t *testing.T) {
		id := createTestOrigin(t, db, "__test__del-blocked", 3001)

		var locationID, categoryID int
		require.NoError(t, db.QueryRow("SELECT id FROM locations LIMIT 1").Scan(&locationID))
		require.NoError(t, db.QueryRow("SELECT id FROM item_category LIMIT 1").Scan(&categoryID))

		_, err := db.Exec(
			"INSERT INTO items (location_id, item_category_id, origin_id, item_serial) VALUES ($1, $2, $3, '__TEST__orig-block-item')",
			locationID, categoryID, id,
		)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/origins/%d", id), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		var resp map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Cannot delete origin with assigned equipment", resp["error"])
	})

	t.Run("non-existent id returns 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/origins/999999999", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("non-numeric id returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/origins/abc", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
