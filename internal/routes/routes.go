package routes

import (
	"log"
	"net/http"
	"os"
	"warehouse/internal/assets"
	"warehouse/internal/locations"
	UserRepository "warehouse/internal/repository/user"
	"warehouse/internal/stocks"
	"warehouse/internal/transfers"
	"warehouse/internal/users"
	"warehouse/pkg/auditlog"
	"warehouse/pkg/security"

	"warehouse/internal/repository"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(router *gin.Engine, repo *repository.Repository, auditLog *auditlog.Auditlog) {
	security.RegisterRoutes(router, repo)
	assets.RegisterRoutes(router, repo, auditLog)
	stocks.RegisterRoutes(router, repo, auditLog)
	transfers.RegisterRoutes(router, repo, auditLog)
	locations.RegisterRoutes(router, repo)
}

func RegisterProtectedRoutes(router *gin.Engine, repo *repository.Repository, auditLog *auditlog.Auditlog, userRepo *UserRepository.UserRepository) {
	protectedRoutes := router.Group("")
	protectedRoutes.Use(security.JWTMiddleware())

	users.RegisterRoutes(protectedRoutes, repo.DB, userRepo)
}

func RegisterUtilityRoutes(router *gin.Engine) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		log.Println("Health check successful")
	})

	openapiFilePath := "./docs/index.html"
	if _, err := os.Stat(openapiFilePath); err == nil {
		router.GET("/openapi.html", func(c *gin.Context) {
			c.File(openapiFilePath)
		})
		log.Println("Route docs/index.html registered successfully.")
	} else {
		log.Printf("Warning: %s not found. Route /openapi.html will not be registered.\n", openapiFilePath)
	}
}
