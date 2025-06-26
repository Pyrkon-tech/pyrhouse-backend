package security

import (
	"net/http"
	"strconv"
	"time"
	"warehouse/internal/rate_limiter"
	"warehouse/internal/repository"

	"github.com/gin-gonic/gin"
)

type LoginHandler struct {
	repository  *repository.Repository
	rateLimiter *rate_limiter.RateLimiter
}

func NewLoginHandler(repository *repository.Repository) *LoginHandler {
	return &LoginHandler{
		repository:  repository,
		rateLimiter: rate_limiter.NewRateLimiter(7, 5*time.Minute),
	}
}

func (l *LoginHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/auth", l.LoginHandler())
}

func (l *LoginHandler) LoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if !l.rateLimiter.IsAllowed(clientIP) {
			remaining := l.rateLimiter.GetRemainingRequests(clientIP)
			c.Header("X-RateLimit-Limit", "5")
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

		user, err := AuthenticateUser(req.Username, req.Password, l.repository)
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
