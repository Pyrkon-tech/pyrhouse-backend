package locations

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
	"warehouse/internal/models"
	"warehouse/internal/repository"
)

func testDBURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://postgres:pyrpyr@localhost:15432/pyrhouse_test?sslmode=disable"
}

func setupLocationsTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("postgres", testDBURL())
	if err != nil {
		t.Skipf("Test database not available: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Skipf("Cannot connect to test database: %v", err)
	}
	cleanup := func() {
		_, _ = db.Exec("DELETE FROM items WHERE item_serial LIKE '__TEST__%'")
		_, _ = db.Exec("DELETE FROM locations WHERE name LIKE '__TEST__%'")
		_ = db.Close()
	}
	return db, cleanup
}

func newLocationsRouter(h *LocationHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("role", "admin"); c.Next() })
	h.RegisterPublicRoutes(r)
	h.RegisterRoutes(r.Group("/"))
	return r
}

func locJSONBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

func createTestLocation(t *testing.T, db *sql.DB, name string) int {
	t.Helper()
	var id int
	err := db.QueryRow(
		"INSERT INTO locations (name) VALUES ($1) RETURNING id", name,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestLocations_Create covers POST /locations
func TestLocations_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupLocationsTestDB(t)
	defer cleanup()

	repo := NewLocationRepository(repository.NewRepository(db))
	router := newLocationsRouter(NewLocationHandler(repo))

	t.Run("valid location is created", func(t *testing.T) {
		payload := map[string]any{"name": "__TEST__Create_Alpha"}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/locations", locJSONBody(t, payload))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var loc models.Location
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loc))
		assert.NotZero(t, loc.ID)
		assert.Equal(t, "__TEST__Create_Alpha", loc.Name)
	})

	t.Run("location with pavilion is created", func(t *testing.T) {
		pavilion := "Hala A"
		payload := map[string]any{"name": "__TEST__Create_Beta", "pavilion": pavilion}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/locations", locJSONBody(t, payload))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var loc models.Location
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loc))
		require.NotNil(t, loc.Pavilion)
		assert.Equal(t, pavilion, *loc.Pavilion)
	})
}

// TestLocations_GetList covers GET /locations
func TestLocations_GetList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupLocationsTestDB(t)
	defer cleanup()

	createTestLocation(t, db, "__TEST__List_One")
	createTestLocation(t, db, "__TEST__List_Two")

	repo := NewLocationRepository(repository.NewRepository(db))
	router := newLocationsRouter(NewLocationHandler(repo))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/locations", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var locs []models.Location
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &locs))

	names := make([]string, len(locs))
	for i, l := range locs {
		names[i] = l.Name
	}
	assert.Contains(t, names, "__TEST__List_One")
	assert.Contains(t, names, "__TEST__List_Two")
}

// TestLocations_GetSingle covers GET /locations/:id
func TestLocations_GetSingle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupLocationsTestDB(t)
	defer cleanup()

	id := createTestLocation(t, db, "__TEST__Single_One")

	repo := NewLocationRepository(repository.NewRepository(db))
	router := newLocationsRouter(NewLocationHandler(repo))

	tests := []struct {
		name       string
		id         string
		wantStatus int
		wantName   string
	}{
		{
			name:       "existing location",
			id:         fmt.Sprintf("%d", id),
			wantStatus: http.StatusOK,
			wantName:   "__TEST__Single_One",
		},
		{
			name:       "non-existent location returns 404",
			id:         "999999999",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/locations/"+tt.id, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantName != "" {
				var loc models.Location
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loc))
				assert.Equal(t, tt.wantName, loc.Name)
				assert.Equal(t, id, loc.ID)
			}
		})
	}
}

// TestLocations_Update covers PATCH /locations/:id
func TestLocations_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupLocationsTestDB(t)
	defer cleanup()

	id := createTestLocation(t, db, "__TEST__Update_Original")

	repo := NewLocationRepository(repository.NewRepository(db))
	router := newLocationsRouter(NewLocationHandler(repo))

	tests := []struct {
		name       string
		payload    map[string]any
		wantStatus int
		checkFn    func(t *testing.T, body []byte)
	}{
		{
			name:       "update name",
			payload:    map[string]any{"name": "__TEST__Update_Renamed"},
			wantStatus: http.StatusOK,
			checkFn: func(t *testing.T, body []byte) {
				var loc models.Location
				require.NoError(t, json.Unmarshal(body, &loc))
				assert.Equal(t, "__TEST__Update_Renamed", loc.Name)
			},
		},
		{
			name:       "update pavilion",
			payload:    map[string]any{"pavilion": "Hala B"},
			wantStatus: http.StatusOK,
			checkFn: func(t *testing.T, body []byte) {
				var loc models.Location
				require.NoError(t, json.Unmarshal(body, &loc))
				require.NotNil(t, loc.Pavilion)
				assert.Equal(t, "Hala B", *loc.Pavilion)
			},
		},
		{
			name:       "empty payload returns 400",
			payload:    map[string]any{},
			wantStatus: http.StatusBadRequest,
			checkFn:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/locations/%d", id), locJSONBody(t, tt.payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.checkFn != nil {
				tt.checkFn(t, w.Body.Bytes())
			}
		})
	}
}

// TestLocations_Delete covers DELETE /locations/:id
func TestLocations_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupLocationsTestDB(t)
	defer cleanup()

	repo := NewLocationRepository(repository.NewRepository(db))
	router := newLocationsRouter(NewLocationHandler(repo))

	t.Run("empty location is deleted", func(t *testing.T) {
		id := createTestLocation(t, db, "__TEST__Delete_Empty")

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/locations/%d", id), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// confirm gone from DB
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM locations WHERE id = $1", id).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("location with assigned item is blocked with 409", func(t *testing.T) {
		id := createTestLocation(t, db, "__TEST__Delete_WithItem")

		// get any valid category id
		var categoryID int
		err := db.QueryRow("SELECT id FROM item_category LIMIT 1").Scan(&categoryID)
		require.NoError(t, err, "need at least one item_category in DB")

		_, err = db.Exec(
			"INSERT INTO items (location_id, item_category_id, item_serial) VALUES ($1, $2, '__TEST__item-block-001')",
			id, categoryID,
		)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/locations/%d", id), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var resp map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Cannot delete location, it has assigned items", resp["error"])
	})

	t.Run("non-existent location returns 500", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/locations/999999999", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
