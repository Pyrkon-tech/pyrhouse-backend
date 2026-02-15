package middleware

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthStatus represents the application health status
type HealthStatus struct {
	Status      string    `json:"status"`
	LastChecked time.Time `json:"last_checked"`
	Uptime      string    `json:"uptime"`
	Version     string    `json:"version"`
}

var (
	healthStatus = HealthStatus{
		Status:      "ok",
		LastChecked: time.Now(),
		Uptime:      "0s",
		Version:     "1.0.0",
	}
	healthMutex      sync.RWMutex
	startTime        = time.Now()
	lastResponse     []byte
	lastResponseTime time.Time
	cacheDuration    = 5 * time.Second
)

// HealthCheckMiddleware provides an application health check endpoint.
func HealthCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		healthMutex.RLock()
		if time.Since(lastResponseTime) < cacheDuration && lastResponse != nil {
			cached := make([]byte, len(lastResponse))
			copy(cached, lastResponse)
			healthMutex.RUnlock()
			c.Data(http.StatusOK, "application/json", cached)
			return
		}
		healthMutex.RUnlock()

		healthMutex.Lock()
		// Double-check after acquiring write lock
		if time.Since(lastResponseTime) < cacheDuration && lastResponse != nil {
			cached := make([]byte, len(lastResponse))
			copy(cached, lastResponse)
			healthMutex.Unlock()
			c.Data(http.StatusOK, "application/json", cached)
			return
		}

		healthStatus.Uptime = time.Since(startTime).String()
		healthStatus.LastChecked = time.Now()

		response, err := json.Marshal(healthStatus)
		if err != nil {
			healthMutex.Unlock()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal health status"})
			return
		}
		lastResponse = response
		lastResponseTime = time.Now()
		healthMutex.Unlock()

		c.Data(http.StatusOK, "application/json", response)
	}
}

// UpdateHealthStatus updates the application health status
func UpdateHealthStatus(status string) {
	healthMutex.Lock()
	defer healthMutex.Unlock()

	healthStatus.Status = status
	healthStatus.LastChecked = time.Now()
	lastResponse = nil // Invalidate cache
}

// SetVersion sets the application version
func SetVersion(version string) {
	healthMutex.Lock()
	defer healthMutex.Unlock()

	healthStatus.Version = version
	lastResponse = nil // Invalidate cache
}
