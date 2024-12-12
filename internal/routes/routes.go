package routes

import (
	"log"
	"net/http"
	"os"
	"warehouse/internal/container"
	"warehouse/pkg/security"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(router *gin.Engine, container *container.Container) {
	container.LoginHandler.RegisterRoutes(router)
	container.AssetHandler.RegisterRoutes(router)
	container.StockHandler.RegisterRoutes(router)
	container.TransferHandler.RegisterRoutes(router)
	container.LocationHandler.RegisterRoutes(router)
	container.ItemHandler.RegisterRoutes(router)
}

func RegisterProtectedRoutes(router *gin.Engine, container *container.Container) {
	protectedRoutes := router.Group("")
	protectedRoutes.Use(security.JWTMiddleware())

	container.UserHandler.RegisterRoutes(protectedRoutes)
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
