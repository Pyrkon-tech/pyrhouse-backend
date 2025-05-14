package security

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"warehouse/internal/rate_limiter"
	"warehouse/internal/repository"

	"github.com/gin-gonic/gin"
)

type LoginHandler struct {
	repo        *repository.Repository
	rateLimiter *rate_limiter.RateLimiter
}

func NewLoginHandler(r *repository.Repository) *LoginHandler {
	return &LoginHandler{
		repo:        r,
		rateLimiter: rate_limiter.NewRateLimiter(10, 5*time.Minute), // 10 prób na 5 minut
	}
}

func (l *LoginHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/auth", l.LoginHandler())
}

func (l *LoginHandler) LoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Pobierz IP z nagłówka X-Forwarded-For lub X-Real-IP, jeśli są dostępne
		clientIP := c.GetHeader("X-Forwarded-For")
		if clientIP == "" {
			clientIP = c.GetHeader("X-Real-IP")
		}
		if clientIP == "" {
			clientIP = c.ClientIP()
		}

		// Jeśli mamy kilka IP (np. z X-Forwarded-For), weź pierwsze
		if strings.Contains(clientIP, ",") {
			clientIP = strings.Split(clientIP, ",")[0]
		}

		// Sprawdź czy IP nie jest prywatne
		if isPrivateIP(clientIP) {
			// Jeśli IP jest prywatne, użyj kombinacji IP i User-Agent
			userAgent := c.GetHeader("User-Agent")
			clientIP = clientIP + ":" + userAgent
		}

		if !l.rateLimiter.IsAllowed(clientIP) {
			remaining := l.rateLimiter.GetRemainingRequests(clientIP)
			c.Header("X-RateLimit-Limit", "10")
			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.Header("X-RateLimit-Reset", time.Now().Add(5*time.Minute).Format(time.RFC3339))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":     "Przekroczono limit prób logowania. Spróbuj ponownie później.",
				"remaining": remaining,
				"reset_at":  time.Now().Add(5 * time.Minute).Format(time.RFC3339),
			})
			return
		}

		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
			return
		}

		user, err := AuthenticateUser(req.Username, req.Password, l.repo)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		token, err := GenerateJWT(strconv.Itoa(user.ID), string(user.Role), user.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": token})
	}
}

// isPrivateIP sprawdza czy IP jest prywatne
func isPrivateIP(ip string) bool {
	// Sprawdź czy IP zaczyna się od znanych prefiksów prywatnych
	privatePrefixes := []string{
		"10.",
		"172.16.",
		"172.17.",
		"172.18.",
		"172.19.",
		"172.20.",
		"172.21.",
		"172.22.",
		"172.23.",
		"172.24.",
		"172.25.",
		"172.26.",
		"172.27.",
		"172.28.",
		"172.29.",
		"172.30.",
		"172.31.",
		"192.168.",
		"127.",
		"169.254.",
		"::1",
		"fc00::",
		"fe80::",
	}

	for _, prefix := range privatePrefixes {
		if strings.HasPrefix(ip, prefix) {
			return true
		}
	}
	return false
}
