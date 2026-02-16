package di

import (
	"database/sql"
	"log"
	"warehouse/internal/auditlog"
	auditLogRepo "warehouse/internal/auditlog"
	"warehouse/internal/config"
	"warehouse/internal/equipment_requests"
	"warehouse/internal/integrations/googlesheets"
	"warehouse/internal/inventory/assets"
	"warehouse/internal/inventory/category"
	"warehouse/internal/inventory/items"
	"warehouse/internal/inventory/stocks"
	"warehouse/internal/inventory/transfers"
	"warehouse/internal/locations"
	"warehouse/internal/oauth"
	"warehouse/internal/repository"
	"warehouse/internal/security"
	"warehouse/internal/service_desk"
	"warehouse/internal/users"
)

type Container struct {
	Repository               *repository.Repository
	AuditLog                 *auditlog.Auditlog
	LoginHandler             *security.LoginHandler
	AssetHandler             *assets.ItemHandler
	StockHandler             *stocks.StockHandler
	LocationHandler          *locations.LocationHandler
	TransferHandler          *transfers.TransferHandler
	UserHandler              *users.UsersHandler
	ItemHandler              *items.ItemHandler
	GoogleSheetsHandler      *googlesheets.GoogleSheetsHandler
	ItemCategoryHandler      *category.ItemCategoryHandler
	ServiceDeskHandler       *service_desk.Handler
	DiscordHandler           *security.DiscordHandler
	EquipmentRequestHandler  *equipment_requests.Handler
	EquipmentRequestScheduler *equipment_requests.Scheduler
}

func NewAppContainer(db *sql.DB, cfg *config.Config) *Container {
	repo := repository.NewRepository(db)
	auditLogRepo := auditLogRepo.NewRepository(repo)
	assetRepo := assets.NewRepository(repo)
	userRepo := users.NewRepository(repo)
	auditLog := auditlog.NewAuditLog(auditLogRepo)
	userHandler := users.NewHandler(userRepo)
	loginHandler := security.NewLoginHandler(repo)
	assetHandler := assets.NewAssetHandler(repo, assetRepo, auditLog)
	stockRepo := stocks.NewRepository(repo)
	stockService := stocks.NewStockService(repo, stockRepo, auditLog)
	stockHandler := stocks.NewStockHandler(repo, stockRepo, auditLog, stockService)
	itemCategoryHandler := category.NewItemCategoryHandler(repo, assetRepo, stockRepo, auditLog)
	locationRepository := locations.NewLocationRepository(repo)
	locationHandler := locations.NewLocationHandler(locationRepository)
	transferRepository := transfers.NewRepository(repo)
	transferHandler := transfers.NewHandler(repo, transferRepository, assetRepo, userRepo, auditLog)
	itemsHandler := items.NewItemHandler(repo, stockRepo, assetRepo, auditLog)
	serviceDeskHandler := service_desk.NewHandler(repo)

	googleSheetsHandler, err := googlesheets.NewGoogleSheetsHandler()
	if err != nil {
		googleSheetsHandler = nil
	}

	var discordHandler *security.DiscordHandler
	if cfg.Discord.ClientID != "" && cfg.Discord.ClientSecret != "" {
		discordOAuth := oauth.NewDiscordOAuth(cfg.Discord)
		discordHandler = security.NewDiscordHandler(discordOAuth, userRepo)
	}

	var equipmentRequestHandler *equipment_requests.Handler
	var equipmentRequestScheduler *equipment_requests.Scheduler
	if cfg.EquipmentRequest.SheetID != "" && googleSheetsHandler != nil {
		categoryRepo := category.NewCategoryRepository(repo)
		equipmentRequestRepo := equipment_requests.NewRepository(repo)
		equipmentRequestService := equipment_requests.NewService(

			googleSheetsHandler.DutyScheduleService,
			categoryRepo,
			equipmentRequestRepo,
			cfg.EquipmentRequest.SheetID,
			cfg.EquipmentRequest.SheetName,
			cfg.EquipmentRequest.FuzzyThreshold,
		)
		equipmentRequestHandler = equipment_requests.NewHandler(equipmentRequestService)

		// Phase 3: Auto-sync scheduler
		if cfg.EquipmentRequest.SyncEnabled {
			equipmentRequestScheduler = equipment_requests.NewScheduler(
				equipmentRequestService,
				cfg.EquipmentRequest.SyncInterval,
			)
			equipmentRequestScheduler.Start()
			log.Printf("[INFO] Equipment request auto-sync enabled (interval: %v)", cfg.EquipmentRequest.SyncInterval)
		} else {
			log.Println("[INFO] Equipment request auto-sync disabled")
		}
	}

	return &Container{
		Repository:                repo,
		AuditLog:                  auditLog,
		LoginHandler:              loginHandler,
		AssetHandler:              assetHandler,
		StockHandler:              stockHandler,
		LocationHandler:           locationHandler,
		TransferHandler:           transferHandler,
		UserHandler:               userHandler,
		ItemHandler:               itemsHandler,
		GoogleSheetsHandler:       googleSheetsHandler,
		ItemCategoryHandler:       itemCategoryHandler,
		ServiceDeskHandler:        serviceDeskHandler,
		DiscordHandler:            discordHandler,
		EquipmentRequestHandler:   equipmentRequestHandler,
		EquipmentRequestScheduler: equipmentRequestScheduler,
	}
}

// Close performs cleanup operations on the container
func (c *Container) Close() {
	// Stop the equipment request scheduler if running
	if c.EquipmentRequestScheduler != nil {
		c.EquipmentRequestScheduler.Stop()
	}
}
