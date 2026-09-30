package releases

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
	"warehouse/internal/repository"
)

func releasesTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:pyrpyr@localhost:15432/pyrhouse_test?sslmode=disable"
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Skipf("Test database not available: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Skipf("Cannot connect to test database: %v", err)
	}
	cleanup := func() {
		_, _ = db.Exec("DELETE FROM releases WHERE created_by IN (SELECT id FROM users WHERE username LIKE '__test_rel_%')")
		_, _ = db.Exec("DELETE FROM items WHERE item_serial LIKE '__TEST__rel%'")
		_, _ = db.Exec("DELETE FROM item_category WHERE label LIKE '__TEST__Rel%'")
		_, _ = db.Exec("DELETE FROM locations WHERE name LIKE '__TEST__Rel%'")
		_, _ = db.Exec("DELETE FROM users WHERE username LIKE '__test_rel_%'")
		_ = db.Close()
	}
	return db, cleanup
}

type releaseFixtures struct {
	userID, originID, assetID int
}

func createReleaseFixtures(t *testing.T, db *sql.DB) releaseFixtures {
	t.Helper()
	var fx releaseFixtures
	require.NoError(t, db.QueryRow(
		"INSERT INTO users (username, password_hash, role) VALUES ('__test_rel_user', 'hash', 'moderator') RETURNING id",
	).Scan(&fx.userID))
	require.NoError(t, db.QueryRow("SELECT id FROM origins ORDER BY id LIMIT 1").Scan(&fx.originID))

	var locID, catID int
	require.NoError(t, db.QueryRow("INSERT INTO locations (name) VALUES ('__TEST__RelLoc') RETURNING id").Scan(&locID))
	require.NoError(t, db.QueryRow(
		"INSERT INTO item_category (item_category, label, pyr_id, category_type) VALUES ('__test__rel_cat', '__TEST__RelCat', 'RL99', 'asset') ON CONFLICT (item_category) DO UPDATE SET label = EXCLUDED.label RETURNING id",
	).Scan(&catID))
	require.NoError(t, db.QueryRow(
		"INSERT INTO items (location_id, item_category_id, item_serial, status, origin_id) VALUES ($1, $2, '__TEST__rel-asset', 'available', $3) RETURNING id",
		locID, catID, fx.originID,
	).Scan(&fx.assetID))
	return fx
}

func newReleasesRouter(db *sql.DB, userID int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("role", "admin")
		c.Set("userID", fmt.Sprintf("%d", userID))
		c.Next()
	})
	repo := repository.NewRepository(db)
	al := auditlog.NewAuditLog(auditlog.NewRepository(repo))
	NewHandler(NewService(NewRepository(repo), repo, al)).RegisterRoutes(r.Group("/"))
	return r
}

func releaseRequest(t *testing.T, router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestReleases_DraftLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := releasesTestDB(t)
	defer cleanup()
	fx := createReleaseFixtures(t, db)
	router := newReleasesRouter(db, fx.userID)

	w := releaseRequest(t, router, http.MethodPost, "/releases", map[string]any{
		"origin_id": fx.originID,
		"assets":    []int{fx.assetID},
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var created map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	// Contract: nullable fields are present (null or value), collections are arrays.
	for _, k := range []string{"origin_label", "created_by_name", "notes", "completed_at"} {
		assert.Contains(t, created, k)
	}
	assert.JSONEq(t, `[]`, string(created["stocks"]))
	var id int
	require.NoError(t, json.Unmarshal(created["id"], &id))

	t.Run("detail", func(t *testing.T) {
		w := releaseRequest(t, router, http.MethodGet, fmt.Sprintf("/releases/%d", id), nil)
		require.Equal(t, http.StatusOK, w.Code)
		var d ReleaseDetail
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &d))
		assert.Equal(t, "draft", d.Status)
		assert.Equal(t, 1, d.Summary.TotalAssets)
		require.Len(t, d.Assets, 1)
		assert.Equal(t, fx.assetID, d.Assets[0].ItemID)
	})

	t.Run("list filtered by status", func(t *testing.T) {
		w := releaseRequest(t, router, http.MethodGet, "/releases?status=draft", nil)
		require.Equal(t, http.StatusOK, w.Code)
		var rows []Release
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rows))
		found := false
		for _, r := range rows {
			found = found || r.ID == id
		}
		assert.True(t, found)
	})

	t.Run("empty create is 400", func(t *testing.T) {
		w := releaseRequest(t, router, http.MethodPost, "/releases", map[string]any{"origin_id": fx.originID})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("delete draft", func(t *testing.T) {
		w := releaseRequest(t, router, http.MethodDelete, fmt.Sprintf("/releases/%d", id), nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	})
}

func TestReleases_UnknownID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := releasesTestDB(t)
	defer cleanup()
	fx := createReleaseFixtures(t, db)
	router := newReleasesRouter(db, fx.userID)

	tests := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"get", http.MethodGet, "/releases/999999999", nil},
		{"update items", http.MethodPut, "/releases/999999999/items", map[string]any{"assets": []int{fx.assetID}}},
		{"confirm", http.MethodPost, "/releases/999999999/confirm", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := releaseRequest(t, router, tt.method, tt.path, tt.body)
			assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
		})
	}
}
