package security

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
	"warehouse/internal/config"
	"warehouse/internal/repository"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func authTestDBURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://postgres:pyrpyr@localhost:15432/pyrhouse_test?sslmode=disable"
}

func setupAuthTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("postgres", authTestDBURL())
	if err != nil {
		t.Skipf("Test database not available: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Skipf("Cannot connect to test database: %v", err)
	}
	cleanup := func() {
		_, _ = db.Exec("DELETE FROM users WHERE username LIKE '__test_auth_%'")
		_ = db.Close()
	}
	return db, cleanup
}

func initTestJWT(t *testing.T) {
	t.Helper()
	require.NoError(t, Initialize(config.JWTConfig{
		Secret:     "test-secret-for-auth-tests",
		Expiration: time.Hour,
	}))
}

func createAuthTestUser(t *testing.T, db *sql.DB, username, password string, active bool) int {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	var id int
	err = db.QueryRow(
		"INSERT INTO users (username, password_hash, role, active) VALUES ($1, $2, 'user', $3) RETURNING id",
		username, string(hash), active,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func newAuthRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := repository.NewRepository(db)
	h := NewLoginHandler(repo)
	h.RegisterRoutes(r)
	return r
}

func authJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

func TestAuth_Login(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupAuthTestDB(t)
	defer cleanup()
	initTestJWT(t)

	createAuthTestUser(t, db, "__test_auth_active", "password123", true)
	createAuthTestUser(t, db, "__test_auth_inactive", "password123", false)

	router := newAuthRouter(db)

	tests := []struct {
		name       string
		payload    map[string]any
		wantStatus int
		checkFn    func(t *testing.T, body map[string]any)
	}{
		{
			name:       "valid credentials return token",
			payload:    map[string]any{"username": "__test_auth_active", "password": "password123"},
			wantStatus: http.StatusOK,
			checkFn: func(t *testing.T, body map[string]any) {
				token, ok := body["token"].(string)
				assert.True(t, ok, "response should contain token")
				assert.NotEmpty(t, token)
			},
		},
		{
			name:       "wrong password returns 401",
			payload:    map[string]any{"username": "__test_auth_active", "password": "wrong"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "non-existent user returns 401",
			payload:    map[string]any{"username": "__test_auth_nobody", "password": "password123"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "inactive account returns 401",
			payload:    map[string]any{"username": "__test_auth_inactive", "password": "password123"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing username returns 400",
			payload:    map[string]any{"password": "password123"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing password returns 400",
			payload:    map[string]any{"username": "__test_auth_active"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty body returns 400",
			payload:    map[string]any{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/auth", authJSON(t, tt.payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.checkFn != nil {
				var body map[string]any
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				tt.checkFn(t, body)
			}
		})
	}
}

func TestAuth_LoginRateLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupAuthTestDB(t)
	defer cleanup()
	initTestJWT(t)

	router := newAuthRouter(db)
	payload := authJSON(t, map[string]any{"username": "__test_auth_nobody", "password": "x"})

	// The rate limiter allows 7 attempts per 5 minutes — exhaust it
	for i := 0; i < 7; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewBuffer(payload.Bytes()))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code, fmt.Sprintf("attempt %d should still be allowed", i+1))
	}

	// 8th request should be rate-limited
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewBuffer(payload.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}
