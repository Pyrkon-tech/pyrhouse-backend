package middleware

import (
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
	healthMutex sync.RWMutex
	startTime   = time.Now()
)

// HealthCheckMiddleware dodaje endpoint do sprawdzania zdrowia aplikacji
func HealthCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		healthMutex.RLock()
		defer healthMutex.RUnlock()

		// Aktualizacja czasu działania
		healthStatus.Uptime = time.Since(startTime).String()
		healthStatus.LastChecked = time.Now()

		c.JSON(http.StatusOK, healthStatus)
	}
}

// UpdateHealthStatus aktualizuje status zdrowia aplikacji
func UpdateHealthStatus(status string) {
	healthMutex.Lock()
	defer healthMutex.Unlock()

	healthStatus.Status = status
	healthStatus.LastChecked = time.Now()
}

// SetVersion ustawia wersję aplikacji
func SetVersion(version string) {
	healthMutex.Lock()
	defer healthMutex.Unlock()

	healthStatus.Version = version
}
