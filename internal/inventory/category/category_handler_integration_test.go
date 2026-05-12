package category

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
	"warehouse/internal/inventory/assets"
	"warehouse/internal/inventory/stocks"
	"warehouse/internal/models"
	"warehouse/internal/repository"
)

func catTestDBURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://postgres:pyrpyr@localhost:15432/pyrhouse_test?sslmode=disable"
}

func setupCategoryTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("postgres", catTestDBURL())
	if err != nil {
		t.Skipf("Test database not available: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Skipf("Cannot connect to test database: %v", err)
	}
	cleanup := func() {
		_, _ = db.Exec("DELETE FROM items WHERE item_serial LIKE '__TEST__%'")
		_, _ = db.Exec("DELETE FROM non_serialized_items WHERE quantity < 0")
		_, _ = db.Exec("DELETE FROM item_category WHERE label LIKE '__TEST__%'")
		_ = db.Close()
	}
	return db, cleanup
}

func newCategoryRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("role", "admin"); c.Next() })

	repo := repository.NewRepository(db)
	ar := assets.NewRepository(repo)
	sr := stocks.NewRepository(repo)
	al := auditlog.NewAuditLog(auditlog.NewRepository(repo))
	h := NewItemCategoryHandler(repo, ar, sr, al)
	h.RegisterRoutes(r.Group("/"))
	return r
}

func catJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

func createTestCategory(t *testing.T, db *sql.DB, label string) int {
	t.Helper()
	var id int
	err := db.QueryRow(
		"INSERT INTO item_category (item_category, label, pyr_id, category_type) VALUES ($1, $2, $3, 'asset') RETURNING id",
		"__test__"+label, label, "T"+label[:1],
	).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestCategory_Create covers POST /assets/categories
func TestCategory_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupCategoryTestDB(t)
	defer cleanup()
	router := newCategoryRouter(db)

	tests := []struct {
		name       string
		payload    map[string]any
		wantStatus int
		checkFn    func(t *testing.T, body []byte)
	}{
		{
			name:       "valid category",
			payload:    map[string]any{"label": "__TEST__Laptop", "type": "asset"},
			wantStatus: http.StatusCreated,
			checkFn: func(t *testing.T, body []byte) {
				var cat models.ItemCategory
				require.NoError(t, json.Unmarshal(body, &cat))
				assert.NotZero(t, cat.ID)
				assert.Equal(t, "__TEST__Laptop", cat.Label)
				assert.NotEmpty(t, cat.PyrID)
			},
		},
		{
			name:       "missing label returns 400",
			payload:    map[string]any{"type": "asset"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing type returns 400 (binding requires alphanum min=1)",
			payload:    map[string]any{"label": "__TEST__Kabel"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/assets/categories", catJSON(t, tt.payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.checkFn != nil {
				tt.checkFn(t, w.Body.Bytes())
			}
		})
	}
}

// TestCategory_GetList covers GET /assets/categories
func TestCategory_GetList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupCategoryTestDB(t)
	defer cleanup()

	createTestCategory(t, db, "__TEST__ListA")
	createTestCategory(t, db, "__TEST__ListB")

	router := newCategoryRouter(db)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/categories", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var cats []models.ItemCategory
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cats))

	labels := make([]string, len(cats))
	for i, c := range cats {
		labels[i] = c.Label
	}
	assert.Contains(t, labels, "__TEST__ListA")
	assert.Contains(t, labels, "__TEST__ListB")
}

// TestCategory_Update covers PATCH /assets/categories/:id
func TestCategory_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupCategoryTestDB(t)
	defer cleanup()

	id := createTestCategory(t, db, "__TEST__UpdateOrig")
	router := newCategoryRouter(db)

	tests := []struct {
		name       string
		id         int
		payload    map[string]any
		wantStatus int
	}{
		{
			name:       "update label",
			id:         id,
			payload:    map[string]any{"label": "__TEST__UpdateRenamed"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty payload returns 400",
			id:         id,
			payload:    map[string]any{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "non-existent id returns 500",
			id:         999999999,
			payload:    map[string]any{"label": "__TEST__Ghost"},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/assets/categories/%d", tt.id), catJSON(t, tt.payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// TestCategory_Delete covers DELETE /assets/categories/:id
func TestCategory_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupCategoryTestDB(t)
	defer cleanup()
	router := newCategoryRouter(db)

	t.Run("empty category is deleted", func(t *testing.T) {
		id := createTestCategory(t, db, "__TEST__DelEmpty")

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/assets/categories/%d", id), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM item_category WHERE id = $1", id).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("category with assigned items is blocked with 409", func(t *testing.T) {
		id := createTestCategory(t, db, "__TEST__DelBlocked")

		var locationID int
		err := db.QueryRow("SELECT id FROM locations LIMIT 1").Scan(&locationID)
		require.NoError(t, err, "need at least one location in DB")

		_, err = db.Exec(
			"INSERT INTO items (location_id, item_category_id, item_serial) VALUES ($1, $2, '__TEST__cat-del-item')",
			locationID, id,
		)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/assets/categories/%d", id), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("non-numeric id returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/assets/categories/abc", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
