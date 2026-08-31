package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthStatus represents the application health status
type HealthStatus struct {
	Status      string    `json:"status"`
	Database    string    `json:"database"`
	LastChecked time.Time `json:"last_checked"`
	Uptime      string    `json:"uptime"`
	Version     string    `json:"version"`
}

const (
	dbStatusOK            = "ok"
	dbStatusUnreachable   = "unreachable"
	dbStatusNotConfigured = "not configured"
)

var (
	healthStatus = HealthStatus{
		Status:      "ok",
		Database:    dbStatusNotConfigured,
		LastChecked: time.Now(),
		Uptime:      "0s",
		Version:     "1.0.0",
	}
	healthMutex      sync.RWMutex
	startTime        = time.Now()
	lastResponse     []byte
	lastResponseTime time.Time
	lastHealthy      bool
	cacheDuration    = 5 * time.Second

	dbCheck        func(ctx context.Context) error
	dbCheckMutex   sync.RWMutex
	dbCheckTimeout = 2 * time.Second
)

// SetDatabaseChecker registers the probe both health endpoints use to report on
// the database. Without it the endpoints answer "not configured" and stay green.
func SetDatabaseChecker(check func(ctx context.Context) error) {
	dbCheckMutex.Lock()
	dbCheck = check
	dbCheckMutex.Unlock()

	invalidateCache()
}

// HealthCheckMiddleware is the liveness probe: it always answers 200 so a
// transient database outage never triggers a restart loop, and reports the
// database state in the payload.
func HealthCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		payload, _ := healthPayload()
		c.Data(http.StatusOK, "application/json", payload)
	}
}

// ReadinessMiddleware is the readiness probe: 503 while the database is
// unreachable, so a load balancer or a deploy check can tell a half-working
// instance from a healthy one.
func ReadinessMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		payload, healthy := healthPayload()

		status := http.StatusOK
		if !healthy {
			status = http.StatusServiceUnavailable
		}

		c.Data(status, "application/json", payload)
	}
}

// healthPayload returns the cached response when it is still fresh, otherwise
// probes the database and rebuilds it.
func healthPayload() ([]byte, bool) {
	healthMutex.RLock()
	if time.Since(lastResponseTime) < cacheDuration && lastResponse != nil {
		cached := make([]byte, len(lastResponse))
		copy(cached, lastResponse)
		healthy := lastHealthy
		healthMutex.RUnlock()
		return cached, healthy
	}
	healthMutex.RUnlock()

	// Probe outside the lock - it does I/O and must not block concurrent readers.
	dbStatus, healthy := probeDatabase()

	healthMutex.Lock()
	defer healthMutex.Unlock()

	healthStatus.Database = dbStatus
	healthStatus.Status = "ok"
	if !healthy {
		healthStatus.Status = "degraded"
	}
	healthStatus.Uptime = time.Since(startTime).String()
	healthStatus.LastChecked = time.Now()

	response, err := json.Marshal(healthStatus)
	if err != nil {
		return []byte(`{"status":"error","database":"unknown"}`), false
	}

	lastResponse = response
	lastResponseTime = time.Now()
	lastHealthy = healthy

	cached := make([]byte, len(response))
	copy(cached, response)

	return cached, healthy
}

func probeDatabase() (string, bool) {
	dbCheckMutex.RLock()
	check := dbCheck
	dbCheckMutex.RUnlock()

	if check == nil {
		return dbStatusNotConfigured, true
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbCheckTimeout)
	defer cancel()

	if err := check(ctx); err != nil {
		return dbStatusUnreachable, false
	}

	return dbStatusOK, true
}

// SetVersion sets the application version
func SetVersion(version string) {
	healthMutex.Lock()
	healthStatus.Version = version
	healthMutex.Unlock()

	invalidateCache()
}

func invalidateCache() {
	healthMutex.Lock()
	defer healthMutex.Unlock()

	lastResponse = nil
	lastResponseTime = time.Time{}
}
