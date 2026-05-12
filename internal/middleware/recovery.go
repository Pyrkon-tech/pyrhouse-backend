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

// TimeoutMiddleware propagates a deadline on the request context so that downstream
// DB queries and HTTP calls respect the timeout via ctx.Done(). Gin does not support
// aborting a running handler from another goroutine without a data race, so the timeout
// is enforced cooperatively: handlers that pass the context to their DB/HTTP calls will
// receive a context.DeadlineExceeded error and return naturally.
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
