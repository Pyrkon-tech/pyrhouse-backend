package security

import (
	"net/http"
	"strconv"
	"warehouse/internal/repository"

	"github.com/gin-gonic/gin"
)

type LoginHandler struct {
	repo *repository.Repository
}

func NewLoginHandler(r *repository.Repository) *LoginHandler {
	return &LoginHandler{repo: r}
}

func (l *LoginHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/auth", l.LoginHandler())
}

func (l *LoginHandler) LoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
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

		token, err := GenerateJWT(strconv.Itoa(user.ID), string(user.Role))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": token})
	}
}
