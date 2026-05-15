package di

import (
	"database/sql"
	"log"
	"warehouse/internal/auditlog"
	auditLogRepo "warehouse/internal/auditlog"
	"warehouse/internal/config"
	"warehouse/internal/budget"
	"warehouse/internal/equipment_requests"
	"warehouse/internal/integrations/googlesheets"
	"warehouse/internal/inventory/assets"
	"warehouse/internal/inventory/category"
	"warehouse/internal/inventory/items"
	"warehouse/internal/inventory/reservations"
	"warehouse/internal/inventory/stocks"
	"warehouse/internal/inventory/transfers"
	"warehouse/internal/locations"
	"warehouse/internal/oauth"
	"warehouse/internal/origins"
	"warehouse/internal/repository"
	"warehouse/internal/security"
	"warehouse/internal/dispatch"
	"warehouse/internal/releases"
	"warehouse/internal/scheduling"
	"warehouse/internal/search"
	"warehouse/internal/service_desk"
	"warehouse/internal/settings"
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
	DispatchHandler          *dispatch.Handler
	DispatchBroadcaster      *dispatch.Broadcaster
	DispatchSlotWatcher      *dispatch.SlotWatcher
	ReleaseHandler           *releases.Handler
	ServiceDeskHandler       *service_desk.Handler
	DiscordHandler           *security.DiscordHandler
	GoogleHandler            *security.GoogleHandler
	EquipmentRequestHandler   *equipment_requests.Handler
	EquipmentRequestScheduler *equipment_requests.Scheduler
	BudgetHandler             *budget.Handler
	OriginHandler             *origins.Handler
	OriginService             *origins.Service
	SchedulingHandler         *scheduling.Handler
	SearchHandler             *search.Handler
	SettingsHandler           *settings.Handler
	SettingsRepo              *settings.Repository
	ReservationHandler        *reservations.Handler
}

func NewAppContainer(db *sql.DB, cfg *config.Config) *Container {
	repo := repository.NewRepository(db)
	auditLogRepo := auditLogRepo.NewRepository(repo)
	assetRepo := assets.NewRepository(repo)
	userRepo := users.NewRepository(repo)
	auditLog := auditlog.NewAuditLog(auditLogRepo)
	settingsRepo := settings.NewRepository(repo)
	settingsHandler := settings.NewHandler(settingsRepo)
	originRepo := origins.NewRepository(repo)
	originService := origins.NewService(originRepo)
	originHandler := origins.NewHandler(originService)
	userHandler := users.NewHandler(userRepo)
	loginHandler := security.NewLoginHandler(repo)
	assetHandler := assets.NewAssetHandler(repo, assetRepo, auditLog, originService)
	stockRepo := stocks.NewRepository(repo)
	stockService := stocks.NewStockService(repo, stockRepo, auditLog)
	stockHandler := stocks.NewStockHandler(repo, stockRepo, auditLog, stockService, originService)
	itemCategoryHandler := category.NewItemCategoryHandler(repo, assetRepo, stockRepo, auditLog)
	locationRepository := locations.NewLocationRepository(repo)
	locationHandler := locations.NewLocationHandler(locationRepository)
	transferRepository := transfers.NewRepository(repo)
	transferHandler := transfers.NewHandler(repo, transferRepository, assetRepo, userRepo, auditLog)
	itemsHandler := items.NewItemHandler(repo, stockRepo, assetRepo, auditLog)
	dispatchRepo := dispatch.NewRepository(repo)
	dispatchBroadcaster := dispatch.NewBroadcaster(db)
	dispatchSlotWatcher := dispatch.NewSlotWatcher(db, dispatchBroadcaster)
	dispatchSlotWatcher.Start()
	dispatchHandler := dispatch.NewHandler(dispatchRepo, dispatchBroadcaster)
	releaseRepo := releases.NewRepository(repo)
	releaseService := releases.NewService(releaseRepo, repo, auditLog)
	releaseHandler := releases.NewHandler(releaseService)
	searchRepo := search.NewRepository(repo)
	searchHandler := search.NewHandler(searchRepo)
	serviceDeskHandler := service_desk.NewHandler(repo)

	googleSheetsHandler, err := googlesheets.NewGoogleSheetsHandler()
	if err != nil {
		googleSheetsHandler = nil
	}

	schedulingRepo := scheduling.NewRepository(repo)
	schedulingService := scheduling.NewService(schedulingRepo, googleSheetsHandler, settingsRepo)
	schedulingHandler := scheduling.NewHandler(schedulingService)

	var discordHandler *security.DiscordHandler
	if cfg.Discord.ClientID != "" && cfg.Discord.ClientSecret != "" {
		discordOAuth := oauth.NewDiscordOAuth(cfg.Discord)
		discordHandler = security.NewDiscordHandler(discordOAuth, userRepo)
	}

	var googleHandler *security.GoogleHandler
	if cfg.Google.ClientID != "" && cfg.Google.ClientSecret != "" {
		googleOAuth := oauth.NewGoogleOAuth(cfg.Google)
		googleHandler = security.NewGoogleHandler(googleOAuth, userRepo)
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
			settingsRepo,
			cfg.EquipmentRequest.SheetID,
			cfg.EquipmentRequest.SheetName,
			cfg.EquipmentRequest.FuzzyThreshold,
		)

		// Phase 4: Wire quest → transfer integration
		equipmentRequestService.SetTransferCreator(transferHandler.Service)
		transferHandler.Service.RegisterStatusCallback(equipmentRequestService)

		// Wire stock changes → SSE broadcast
		stockService.OnStockChanged = equipmentRequestService.BroadcastStocksChanged

		// Wire dispatch SSE hooks
		equipmentRequestService.SetDispatchHooks(
			dispatchBroadcaster.BroadcastTransferDispatched,
			dispatchBroadcaster.BroadcastTransferEnded,
		)

		equipmentRequestHandler = equipment_requests.NewHandler(equipmentRequestService)

		// Phase 3: Auto-sync scheduler
		if cfg.EquipmentRequest.SyncEnabled {
			equipmentRequestScheduler = equipment_requests.NewScheduler(
				equipmentRequestService,
				cfg.EquipmentRequest.SyncInterval,
			)
			equipmentRequestScheduler.Start()
			equipmentRequestHandler.SetScheduler(equipmentRequestScheduler)
			log.Printf("[INFO] Equipment request auto-sync enabled (interval: %v)", cfg.EquipmentRequest.SyncInterval)
		} else {
			log.Println("[INFO] Equipment request auto-sync disabled")
		}
	}

	reservationRepo := reservations.NewRepository(repo)
	reservationService := reservations.NewService(reservationRepo, assetRepo, repo, originService, auditLog)
	reservationHandler := reservations.NewHandler(reservationService)

	// Budget handler — always wired (reads from DB; gracefully returns empty if no quests yet)
	budgetRepo := budget.NewRepository(repo)
	var budgetSheetReader budget.SheetReader
	if googleSheetsHandler != nil {
		budgetSheetReader = googleSheetsHandler.DutyScheduleService
	}
	budgetService := budget.NewService(budgetRepo, budgetSheetReader, settingsRepo, cfg.EquipmentRequest.SheetID)
	budgetHandler := budget.NewHandler(budgetService)

	// Hook: sync Cennik prices automatically after each quest sync (only when sheet reader is configured)
	if equipmentRequestScheduler != nil && budgetSheetReader != nil {
		equipmentRequestScheduler.SetPostSyncHook(budgetService.SyncPricesFromSheetCtx)
		log.Println("[INFO] Budget price sync hooked into equipment request auto-sync")
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
		DispatchHandler:           dispatchHandler,
		DispatchBroadcaster:       dispatchBroadcaster,
		DispatchSlotWatcher:       dispatchSlotWatcher,
		ReleaseHandler:            releaseHandler,
		ServiceDeskHandler:        serviceDeskHandler,
		DiscordHandler:            discordHandler,
		GoogleHandler:             googleHandler,
		EquipmentRequestHandler:   equipmentRequestHandler,
		EquipmentRequestScheduler: equipmentRequestScheduler,
		BudgetHandler:             budgetHandler,
		OriginHandler:             originHandler,
		OriginService:             originService,
		SchedulingHandler:         schedulingHandler,
		SearchHandler:             searchHandler,
		SettingsHandler:           settingsHandler,
		SettingsRepo:              settingsRepo,
		ReservationHandler:        reservationHandler,
	}
}

// Close performs cleanup operations on the container
func (c *Container) Close() {
	if c.EquipmentRequestScheduler != nil {
		c.EquipmentRequestScheduler.Stop()
	}
	if c.DispatchSlotWatcher != nil {
		c.DispatchSlotWatcher.Stop()
	}
}
