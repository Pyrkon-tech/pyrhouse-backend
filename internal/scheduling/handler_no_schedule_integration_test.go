package scheduling

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"warehouse/internal/repository"
)

// Without an active schedule, what hangs off it is "not found", like the schedule itself — not a server error.
func TestSchedule_NoActiveSchedule_IsNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:pyrpyr@localhost:15432/pyrhouse_test?sslmode=disable"
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Skipf("Test database not available: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Skipf("Cannot connect to test database: %v", err)
	}
	var active int
	if err := db.QueryRow("SELECT count(*) FROM schedules WHERE status = 'active'").Scan(&active); err != nil || active > 0 {
		t.Skip("The test database has an active schedule; this checks the case without one")
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("role", "admin"); c.Next() })
	NewHandler(NewService(NewRepository(repository.NewRepository(db)), nil, nil)).RegisterRoutes(router.Group("/"))

	for _, path := range []string{"/schedule", "/schedule/volunteers"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		assert.Equal(t, http.StatusNotFound, w.Code, path)
	}
}
