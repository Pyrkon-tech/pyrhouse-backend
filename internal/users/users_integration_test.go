package users

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"warehouse/internal/models"
	"warehouse/internal/repository"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func usersTestDBURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://postgres:pyrpyr@localhost:15432/pyrhouse_test?sslmode=disable"
}

func setupUsersTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("postgres", usersTestDBURL())
	if err != nil {
		t.Skipf("Test database not available: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Skipf("Cannot connect to test database: %v", err)
	}
	cleanup := func() {
		_, _ = db.Exec("DELETE FROM users WHERE username LIKE '__test_usr_%'")
		_ = db.Close()
	}
	return db, cleanup
}

// newUsersRouter builds a router with a fake auth middleware that injects the
// provided role and userID into the Gin context, bypassing JWT validation.
func newUsersRouter(db *sql.DB, role string, authUserID int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("role", role)
		c.Set("userID", fmt.Sprintf("%d", authUserID))
		c.Next()
	})
	repo := repository.NewRepository(db)
	h := NewHandler(NewRepository(repo))
	h.RegisterRoutes(r.Group("/"))
	h.RegisterPublicRoutes(r)
	return r
}

func toJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

// insertUser directly inserts a test user and returns its ID.
func insertUser(t *testing.T, db *sql.DB, username, role string, active bool) int {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	require.NoError(t, err)
	var id int
	err = db.QueryRow(
		`INSERT INTO users (username, password_hash, role, active) VALUES ($1, $2, $3, $4) RETURNING id`,
		username, string(hash), role, active,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

// ── POST /users/register ─────────────────────────────────────────────────────

func TestUsers_PublicRegister(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()
	router := newUsersRouter(db, "user", 0)

	tests := []struct {
		name       string
		payload    map[string]any
		wantStatus int
	}{
		{
			name:       "valid registration creates inactive user",
			payload:    map[string]any{"username": "__test_usr_reg1", "password": "password123"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "duplicate username returns 500",
			payload:    map[string]any{"username": "__test_usr_reg1", "password": "password123"},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "missing username returns 400",
			payload:    map[string]any{"password": "password123"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing password returns 400",
			payload:    map[string]any{"username": "__test_usr_nopwd"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/users/register", toJSON(t, tt.payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}

	// Confirm the registered user is indeed inactive with role=user
	t.Run("registered user is inactive with role user", func(t *testing.T) {
		var active bool
		var role string
		err := db.QueryRow(
			"SELECT active, role FROM users WHERE username = '__test_usr_reg1'",
		).Scan(&active, &role)
		require.NoError(t, err)
		assert.False(t, active)
		assert.Equal(t, "user", role)
	})
}

// ── POST /users (admin registration) ─────────────────────────────────────────

func TestUsers_AdminRegister(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	tests := []struct {
		name       string
		role       string
		payload    map[string]any
		wantStatus int
	}{
		{
			name:       "admin creates active user",
			role:       "admin",
			payload:    map[string]any{"username": "__test_usr_admreg", "password": "password123", "role": "moderator"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "moderator is forbidden",
			role:       "moderator",
			payload:    map[string]any{"username": "__test_usr_modreg", "password": "password123"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "user role is forbidden",
			role:       "user",
			payload:    map[string]any{"username": "__test_usr_userreg", "password": "password123"},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newUsersRouter(db, tt.role, 0)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/users", toJSON(t, tt.payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}

	// Confirm admin-created user is active
	t.Run("admin-created user is active", func(t *testing.T) {
		var active bool
		err := db.QueryRow("SELECT active FROM users WHERE username = '__test_usr_admreg'").Scan(&active)
		require.NoError(t, err)
		assert.True(t, active)
	})
}

// ── GET /users ────────────────────────────────────────────────────────────────

func TestUsers_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	insertUser(t, db, "__test_usr_list1", "user", true)
	insertUser(t, db, "__test_usr_list2", "user", true)

	tests := []struct {
		name       string
		role       string
		wantStatus int
	}{
		{"moderator sees list", "moderator", http.StatusOK},
		{"admin sees list", "admin", http.StatusOK},
		{"dispatcher sees list", "dispatcher", http.StatusOK},
		{"user is forbidden", "user", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newUsersRouter(db, tt.role, 0)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/users", nil)
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantStatus == http.StatusOK {
				var users []models.User
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &users))
				assert.GreaterOrEqual(t, len(users), 2)
			}
		})
	}
}

// ── GET /users/:id ────────────────────────────────────────────────────────────

func TestUsers_GetUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	targetID := insertUser(t, db, "__test_usr_get1", "user", true)
	otherID := insertUser(t, db, "__test_usr_get2", "user", true)

	tests := []struct {
		name       string
		role       string
		authUserID int
		targetID   int
		wantStatus int
	}{
		{"owner can get own profile", "user", targetID, targetID, http.StatusOK},
		{"moderator can get any user", "moderator", otherID, targetID, http.StatusOK},
		{"admin can get any user", "admin", otherID, targetID, http.StatusOK},
		{"user cannot get another user", "user", otherID, targetID, http.StatusForbidden},
		{"non-existent user returns 404", "admin", otherID, 999999999, http.StatusNotFound},
		{"invalid id returns 400", "admin", otherID, -1, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newUsersRouter(db, tt.role, tt.authUserID)
			path := fmt.Sprintf("/users/%d", tt.targetID)
			if tt.name == "invalid id returns 400" {
				path = "/users/abc"
			}
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantStatus == http.StatusOK {
				var u models.User
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &u))
				assert.Equal(t, tt.targetID, u.ID)
				assert.NotEmpty(t, u.Username)
			}
		})
	}
}

// ── PATCH /users/:id ──────────────────────────────────────────────────────────

func TestUsers_UpdateUser_Password(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	userID := insertUser(t, db, "__test_usr_pwd", "user", true)
	otherID := insertUser(t, db, "__test_usr_pwd_other", "user", true)

	tests := []struct {
		name       string
		role       string
		authUserID int
		payload    map[string]any
		wantStatus int
	}{
		{
			name:       "owner can change own password",
			role:       "user",
			authUserID: userID,
			payload:    map[string]any{"password": "newpassword"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "admin can change any password",
			role:       "admin",
			authUserID: otherID,
			payload:    map[string]any{"password": "adminset123"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "other user cannot change password",
			role:       "user",
			authUserID: otherID,
			payload:    map[string]any{"password": "hack"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "password too short returns 400",
			role:       "user",
			authUserID: userID,
			payload:    map[string]any{"password": "abc"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newUsersRouter(db, tt.role, tt.authUserID)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/users/%d", userID), toJSON(t, tt.payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestUsers_UpdateUser_Role(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	userID := insertUser(t, db, "__test_usr_role", "user", true)
	adminID := insertUser(t, db, "__test_usr_role_admin", "admin", true)

	tests := []struct {
		name       string
		role       string
		authUserID int
		payload    map[string]any
		wantStatus int
		checkRole  string
	}{
		{
			name:       "admin can promote user to moderator",
			role:       "admin",
			authUserID: adminID,
			payload:    map[string]any{"role": "moderator"},
			wantStatus: http.StatusOK,
			checkRole:  "moderator",
		},
		{
			name:       "moderator cannot change role",
			role:       "moderator",
			authUserID: adminID,
			payload:    map[string]any{"role": "user"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "user cannot change role",
			role:       "user",
			authUserID: adminID,
			payload:    map[string]any{"role": "admin"},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newUsersRouter(db, tt.role, tt.authUserID)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/users/%d", userID), toJSON(t, tt.payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.checkRole != "" {
				var u models.User
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &u))
				assert.Equal(t, tt.checkRole, string(u.Role))
			}
		})
	}
}

func TestUsers_UpdateUser_Username(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	userID := insertUser(t, db, "__test_usr_uname", "user", true)
	otherID := insertUser(t, db, "__test_usr_uname_other", "user", true)
	insertUser(t, db, "__test_usr_uname_taken", "user", true)

	tests := []struct {
		name       string
		role       string
		authUserID int
		payload    map[string]any
		wantStatus int
	}{
		{
			name:       "owner can change username",
			role:       "user",
			authUserID: userID,
			payload:    map[string]any{"username": "__test_usr_uname_new"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "other user cannot change username",
			role:       "user",
			authUserID: otherID,
			payload:    map[string]any{"username": "__test_usr_uname_hack"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "taken username returns 409",
			role:       "user",
			authUserID: userID,
			payload:    map[string]any{"username": "__test_usr_uname_taken"},
			wantStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newUsersRouter(db, tt.role, tt.authUserID)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/users/%d", userID), toJSON(t, tt.payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestUsers_UpdateUser_Fullname(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	userID := insertUser(t, db, "__test_usr_fname", "user", true)
	otherID := insertUser(t, db, "__test_usr_fname_other", "user", true)

	tests := []struct {
		name       string
		role       string
		authUserID int
		payload    map[string]any
		wantStatus int
	}{
		{
			name:       "owner can change fullname",
			role:       "user",
			authUserID: userID,
			payload:    map[string]any{"fullname": "Jan Kowalski"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "moderator can change fullname",
			role:       "moderator",
			authUserID: otherID,
			payload:    map[string]any{"fullname": "Moderator Set"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "other user cannot change fullname",
			role:       "user",
			authUserID: otherID,
			payload:    map[string]any{"fullname": "Hacker"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "empty fullname returns 400",
			role:       "user",
			authUserID: userID,
			payload:    map[string]any{"fullname": ""},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newUsersRouter(db, tt.role, tt.authUserID)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/users/%d", userID), toJSON(t, tt.payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestUsers_UpdateUser_Active(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	userID := insertUser(t, db, "__test_usr_active", "user", true)
	adminID := insertUser(t, db, "__test_usr_active_admin", "admin", true)
	otherID := insertUser(t, db, "__test_usr_active_other", "user", true)

	tests := []struct {
		name       string
		role       string
		authUserID int
		payload    map[string]any
		wantStatus int
	}{
		{
			name:       "admin can deactivate user",
			role:       "admin",
			authUserID: adminID,
			payload:    map[string]any{"active": false},
			wantStatus: http.StatusOK,
		},
		{
			name:       "admin can reactivate user",
			role:       "admin",
			authUserID: adminID,
			payload:    map[string]any{"active": true},
			wantStatus: http.StatusOK,
		},
		{
			name:       "plain user cannot change active status",
			role:       "user",
			authUserID: otherID,
			payload:    map[string]any{"active": false},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newUsersRouter(db, tt.role, tt.authUserID)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/users/%d", userID), toJSON(t, tt.payload))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// ── POST /users/:id/points ────────────────────────────────────────────────────

func TestUsers_AddPoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	userID := insertUser(t, db, "__test_usr_pts", "user", true)
	adminID := insertUser(t, db, "__test_usr_pts_admin", "admin", true)
	otherID := insertUser(t, db, "__test_usr_pts_other", "user", true)

	tests := []struct {
		name       string
		role       string
		authUserID int
		payload    map[string]any
		wantStatus int
		checkFn    func(t *testing.T, body map[string]any)
	}{
		{
			name:       "admin adds points",
			role:       "admin",
			authUserID: adminID,
			payload:    map[string]any{"points": 10},
			wantStatus: http.StatusOK,
			checkFn: func(t *testing.T, body map[string]any) {
				assert.Equal(t, float64(10), body["points"])
			},
		},
		{
			name:       "admin adds more points (cumulative)",
			role:       "admin",
			authUserID: adminID,
			payload:    map[string]any{"points": 5},
			wantStatus: http.StatusOK,
			checkFn: func(t *testing.T, body map[string]any) {
				assert.Equal(t, float64(15), body["points"])
			},
		},
		{
			name:       "non-admin is forbidden",
			role:       "user",
			authUserID: otherID,
			payload:    map[string]any{"points": 100},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "missing points returns 400",
			role:       "admin",
			authUserID: adminID,
			payload:    map[string]any{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newUsersRouter(db, tt.role, tt.authUserID)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/users/%d/points", userID), toJSON(t, tt.payload))
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

// ── DELETE /users/:id ─────────────────────────────────────────────────────────

func TestUsers_DeleteUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	adminID := insertUser(t, db, "__test_usr_del_admin", "admin", true)
	otherID := insertUser(t, db, "__test_usr_del_other", "user", true)

	t.Run("non-admin is forbidden", func(t *testing.T) {
		targetID := insertUser(t, db, "__test_usr_del_target1", "user", true)
		router := newUsersRouter(db, "user", otherID)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/users/%d", targetID), nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("admin deletes user", func(t *testing.T) {
		targetID := insertUser(t, db, "__test_usr_del_target2", "user", true)
		router := newUsersRouter(db, "admin", adminID)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/users/%d", targetID), nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var count int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM users WHERE id = $1", targetID).Scan(&count))
		assert.Equal(t, 0, count)
	})

	t.Run("non-existent user returns 500", func(t *testing.T) {
		router := newUsersRouter(db, "admin", adminID)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/users/999999999", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		router := newUsersRouter(db, "admin", adminID)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/users/abc", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Confirm admin cannot be deleted by another admin if they own transfers — would be 409
	// (not tested here as it requires transfer setup; tested at repository level in repo_test)
	_ = otherID
}

// ── Repository: Discord & Google link ────────────────────────────────────────

func TestRepository_LinkDiscord(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	userID := insertUser(t, db, "__test_usr_disc", "user", true)
	repo := NewRepository(repository.NewRepository(db))

	t.Run("link discord to user", func(t *testing.T) {
		err := repo.LinkDiscord(userID, "disc_id_001", "DiscordUser#0001", "https://cdn.discordapp.com/avatar.png")
		require.NoError(t, err)

		user, err := repo.GetUser(userID)
		require.NoError(t, err)
		require.NotNil(t, user.DiscordID)
		assert.Equal(t, "disc_id_001", *user.DiscordID)
		assert.Equal(t, "DiscordUser#0001", *user.DiscordUsername)
	})

	t.Run("find user by discord id", func(t *testing.T) {
		found, err := repo.FindUserByDiscordID("disc_id_001")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, userID, found.ID)
	})

	t.Run("find non-existent discord id returns nil", func(t *testing.T) {
		found, err := repo.FindUserByDiscordID("disc_id_nobody")
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("update discord info", func(t *testing.T) {
		err := repo.UpdateDiscordInfo(userID, "DiscordUser#0002", "https://cdn.discordapp.com/new_avatar.png")
		require.NoError(t, err)

		user, err := repo.GetUser(userID)
		require.NoError(t, err)
		assert.Equal(t, "DiscordUser#0002", *user.DiscordUsername)
		assert.Equal(t, "https://cdn.discordapp.com/new_avatar.png", *user.AvatarURL)
	})
}

func TestRepository_MergeDiscordAccount(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	repo := NewRepository(repository.NewRepository(db))

	targetID := insertUser(t, db, "__test_usr_merge_target", "user", true)
	sourceID := insertUser(t, db, "__test_usr_merge_source", "user", true)

	// Give the source user a Discord account
	err := repo.LinkDiscord(sourceID, "disc_merge_001", "DiscordMerge", "https://avatar.png")
	require.NoError(t, err)

	t.Run("merge transfers discord from source to target", func(t *testing.T) {
		sourceDeleted, err := repo.MergeDiscordAccount(targetID, sourceID)
		require.NoError(t, err)

		target, err := repo.GetUser(targetID)
		require.NoError(t, err)
		require.NotNil(t, target.DiscordID)
		assert.Equal(t, "disc_merge_001", *target.DiscordID)

		// sourceDeleted may be false if the source had references; either way the source is deactivated
		if sourceDeleted {
			var count int
			require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM users WHERE id = $1", sourceID).Scan(&count))
			assert.Equal(t, 0, count)
		} else {
			var active bool
			require.NoError(t, db.QueryRow("SELECT active FROM users WHERE id = $1", sourceID).Scan(&active))
			assert.False(t, active)
		}
	})
}

func TestRepository_LinkGoogle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	userID := insertUser(t, db, "__test_usr_goog", "user", true)
	repo := NewRepository(repository.NewRepository(db))

	t.Run("link google to user", func(t *testing.T) {
		err := repo.LinkGoogle(userID, "google_sub_001", "test@pyrkon.pl", "https://lh3.googleusercontent.com/photo.jpg")
		require.NoError(t, err)

		user, err := repo.GetUser(userID)
		require.NoError(t, err)
		require.NotNil(t, user.GoogleID)
		assert.Equal(t, "google_sub_001", *user.GoogleID)
		assert.Equal(t, "test@pyrkon.pl", *user.GoogleEmail)
	})

	t.Run("find user by google id", func(t *testing.T) {
		found, err := repo.FindUserByGoogleID("google_sub_001")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, userID, found.ID)
	})

	t.Run("find non-existent google id returns nil", func(t *testing.T) {
		found, err := repo.FindUserByGoogleID("google_sub_nobody")
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestRepository_CreateDiscordUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	repo := NewRepository(repository.NewRepository(db))

	t.Run("creates new discord user", func(t *testing.T) {
		discordID := "disc_new_001"
		discordUsername := "__test_usr_discord_new"
		avatarURL := "https://avatar.example.com/test.png"
		provider := "discord"

		newUser := &models.User{
			Username:        discordUsername,
			DiscordID:       &discordID,
			DiscordUsername: &discordUsername,
			AvatarURL:       &avatarURL,
			AuthProvider:    &provider,
			Role:            "user",
			Active:          false,
		}

		created, err := repo.CreateDiscordUser(newUser)
		require.NoError(t, err)
		assert.Greater(t, created.ID, 0)
		assert.Equal(t, discordID, *created.DiscordID)
	})

	t.Run("username collision appends discord id suffix", func(t *testing.T) {
		// Insert a user with the name we'll try to create via discord
		insertUser(t, db, "__test_usr_discord_collision", "user", true)

		discordID := "disc_collision_001"
		discordUsername := "__test_usr_discord_collision"
		provider := "discord"

		newUser := &models.User{
			Username:        discordUsername,
			DiscordID:       &discordID,
			DiscordUsername: &discordUsername,
			AuthProvider:    &provider,
			Role:            "user",
			Active:          false,
		}

		created, err := repo.CreateDiscordUser(newUser)
		require.NoError(t, err)
		assert.NotEqual(t, discordUsername, created.Username, "username should have been made unique")
	})
}
