package middleware

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

// RecoveryMiddleware zapewnia odzyskiwanie po awariach i panikach
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Logowanie szczegółów błędu
				log.Printf("[Recovery] Panic recovered: %v\n", err)
				log.Printf("[Recovery] Stack trace: %s\n", debug.Stack())

				// Wysłanie odpowiedzi 500 do klienta
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":   "Internal Server Error",
					"message": "Aplikacja napotkała nieoczekiwany błąd. Został on zarejestrowany i zostanie naprawiony.",
				})
			}
		}()

		c.Next()
	}
}

// TimeoutMiddleware dodaje timeout do żądań
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Tworzymy nowy kontekst z timeoutem
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// Przekazujemy kontekst do żądania
		c.Request = c.Request.WithContext(ctx)

		// Kanał do sygnalizowania zakończenia żądania
		done := make(chan struct{})

		go func() {
			c.Next()
			close(done)
		}()

		// Oczekiwanie na zakończenie żądania lub timeout
		select {
		case <-done:
			// Żądanie zakończone normalnie
			return
		case <-ctx.Done():
			// Timeout - przerwanie żądania
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
				"error":   "Request Timeout",
				"message": "Żądanie przekroczyło dozwolony czas oczekiwania.",
			})
			return
		}
	}
}
