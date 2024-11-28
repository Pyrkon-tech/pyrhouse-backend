package security

import (
	"net/http"
	"strconv"
	"warehouse/internal/repository"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine, repo *repository.Repository) {
	router.POST("/auth", LoginHandler(repo))
}

func LoginHandler(repo *repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
			return
		}

		user, err := AuthenticateUser(req.Username, req.Password, repo)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		token, err := GenerateJWT(strconv.Itoa(user.ID), user.Role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": token})
	}
}
