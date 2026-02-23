package routes

import (
	"log"
	"warehouse/internal/di"
	"warehouse/internal/middleware"
	"warehouse/internal/security"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(router *gin.Engine, container *di.Container) {
	container.LoginHandler.RegisterRoutes(router)
	container.ServiceDeskHandler.RegisterPublicRoutes(router)
	container.UserHandler.RegisterPublicRoutes(router)

	if container.DiscordHandler != nil {
		container.DiscordHandler.RegisterRoutes(router.Group(""))
		log.Println("[Discord OAuth]: Public routes registered")
	}
}

func RegisterProtectedRoutes(router *gin.Engine, container *di.Container) {
	protectedRoutes := router.Group("")
	protectedRoutes.Use(security.JWTMiddleware())

	container.AssetHandler.RegisterRoutes(protectedRoutes)
	container.StockHandler.RegisterRoutes(protectedRoutes)
	container.ItemHandler.RegisterRoutes(protectedRoutes)
	container.ItemCategoryHandler.RegisterRoutes(protectedRoutes)
	container.UserHandler.RegisterRoutes(protectedRoutes)
	container.TransferHandler.RegisterRoutes(protectedRoutes)
	container.LocationHandler.RegisterRoutes(protectedRoutes)
	container.OriginHandler.RegisterRoutes(protectedRoutes)
	container.SettingsHandler.RegisterRoutes(protectedRoutes)
	container.ServiceDeskHandler.RegisterRoutes(protectedRoutes)
	if container.GoogleSheetsHandler != nil {
		container.GoogleSheetsHandler.RegisterRoutes(protectedRoutes)
		log.Println("Google Sheets API routes registered successfully")
	} else {
		log.Println("Google Sheets API routes not registered - handler is nil")
	}

	if container.EquipmentRequestHandler != nil {
		container.EquipmentRequestHandler.RegisterRoutes(protectedRoutes)
		log.Println("[Equipment Requests]: Routes registered successfully")
	} else {
		log.Println("[Equipment Requests]: Routes not registered - handler is nil")
	}

	if container.DiscordHandler != nil {
		container.DiscordHandler.RegisterProtectedRoutes(protectedRoutes)
		log.Println("[Discord OAuth]: Protected routes registered")
	}
}

func RegisterUtilityRoutes(router *gin.Engine) {
	router.GET("/health", middleware.HealthCheckMiddleware())
}
