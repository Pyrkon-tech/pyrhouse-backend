package service_desk

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

func serviceDeskTestDB(t *testing.T) (*sql.DB, func()) {
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
		_, _ = db.Exec("DELETE FROM service_desk_request_comments WHERE request_id IN (SELECT id FROM service_desk_requests WHERE title LIKE '__TEST__%')")
		_, _ = db.Exec("DELETE FROM service_desk_requests WHERE title LIKE '__TEST__%'")
		_, _ = db.Exec("DELETE FROM users WHERE username LIKE '__test_sd_%'")
		_ = db.Close()
	}
	return db, cleanup
}

// newServiceDeskRouter mounts the public routes as-is and the staff routes behind a fake auth middleware.
func newServiceDeskRouter(db *sql.DB, staffID int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(repository.NewRepository(db))
	h.RegisterPublicRoutes(r)
	staff := r.Group("/")
	staff.Use(func(c *gin.Context) {
		c.Set("role", "admin")
		c.Set("userID", fmt.Sprintf("%d", staffID))
		c.Next()
	})
	h.RegisterRoutes(staff)
	return r
}

func sdRequest(t *testing.T, router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
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

func TestServiceDesk_PublicCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := serviceDeskTestDB(t)
	defer cleanup()

	var victimID int
	require.NoError(t, db.QueryRow(
		"INSERT INTO users (username, fullname, password_hash, role) VALUES ('__test_sd_victim', 'Victim', 'hash', 'admin') RETURNING id",
	).Scan(&victimID))
	router := newServiceDeskRouter(db, victimID)

	t.Run("anonymous caller cannot set the author or the assignee", func(t *testing.T) {
		w := sdRequest(t, router, http.MethodPost, "/service-desk/requests", map[string]any{
			"title":         "__TEST__spoof",
			"description":   "x",
			"type":          "other",
			"priority":      "high",
			"created_by":    "Anon",
			"created_by_id": victimID,
			"assigned_to":   victimID,
			"status":        "resolved",
		})
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

		var got map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		assert.Nil(t, got["created_by_user"])
		assert.Nil(t, got["assigned_to_user"])
		assert.Equal(t, "new", got["status"])
		assert.Equal(t, "Anon", got["created_by"])
		assert.Contains(t, got, "location_id") // always present (null)
	})

	tests := []struct {
		name string
		body map[string]any
		want int
	}{
		{"missing title", map[string]any{"type": "other"}, http.StatusBadRequest},
		{"unknown type", map[string]any{"title": "__TEST__t", "type": "bogus"}, http.StatusBadRequest},
		{"unknown priority", map[string]any{"title": "__TEST__p", "type": "other", "priority": "urgent"}, http.StatusBadRequest},
		{"no priority defaults to medium", map[string]any{"title": "__TEST__d", "type": "other"}, http.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := sdRequest(t, router, http.MethodPost, "/service-desk/requests", tt.body)
			assert.Equal(t, tt.want, w.Code, w.Body.String())
			if tt.want == http.StatusCreated {
				var got RequestResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
				assert.Equal(t, PriorityMedium, got.Priority)
			}
		})
	}
}

func TestServiceDesk_StatusAndLookup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := serviceDeskTestDB(t)
	defer cleanup()

	var staffID int
	require.NoError(t, db.QueryRow(
		"INSERT INTO users (username, password_hash, role) VALUES ('__test_sd_staff', 'hash', 'admin') RETURNING id",
	).Scan(&staffID))
	router := newServiceDeskRouter(db, staffID)

	w := sdRequest(t, router, http.MethodPost, "/service-desk/requests", map[string]any{"title": "__TEST__status", "type": "other"})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created RequestResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

	tests := []struct {
		name   string
		method string
		path   string
		body   any
		want   int
	}{
		{"valid status change", http.MethodPut, fmt.Sprintf("/service-desk/requests/%d/status", created.ID), map[string]any{"status": "in_progress"}, http.StatusOK},
		{"invalid status is 400", http.MethodPut, fmt.Sprintf("/service-desk/requests/%d/status", created.ID), map[string]any{"status": "bogus"}, http.StatusBadRequest},
		{"status of unknown request is 404", http.MethodPut, "/service-desk/requests/999999999/status", map[string]any{"status": "closed"}, http.StatusNotFound},
		{"unknown request is 404", http.MethodGet, "/service-desk/requests/999999999", nil, http.StatusNotFound},
		{"list is an array", http.MethodGet, "/service-desk/requests?status=in_progress", nil, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := sdRequest(t, router, tt.method, tt.path, tt.body)
			assert.Equal(t, tt.want, w.Code, w.Body.String())
		})
	}
}
