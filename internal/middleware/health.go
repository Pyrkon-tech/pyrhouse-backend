package middleware

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthStatus reprezentuje status zdrowia aplikacji
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

// HealthCheckMiddleware dodaje endpoint do sprawdzania zdrowia aplikacji
func HealthCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		healthMutex.RLock()
		defer healthMutex.RUnlock()

		// Sprawdź cache
		if time.Since(lastResponseTime) < cacheDuration && lastResponse != nil {
			c.Data(http.StatusOK, "application/json", lastResponse)
			return
		}

		// Aktualizacja czasu działania
		healthStatus.Uptime = time.Since(startTime).String()
		healthStatus.LastChecked = time.Now()

		// Zapisz odpowiedź do cache
		response, _ := json.Marshal(healthStatus)
		lastResponse = response
		lastResponseTime = time.Now()

		c.JSON(http.StatusOK, healthStatus)
	}
}

// UpdateHealthStatus aktualizuje status zdrowia aplikacji
func UpdateHealthStatus(status string) {
	healthMutex.Lock()
	defer healthMutex.Unlock()

	healthStatus.Status = status
	healthStatus.LastChecked = time.Now()
	lastResponse = nil // Invalidate cache
}

// SetVersion ustawia wersję aplikacji
func SetVersion(version string) {
	healthMutex.Lock()
	defer healthMutex.Unlock()

	healthStatus.Version = version
	lastResponse = nil // Invalidate cache
}
