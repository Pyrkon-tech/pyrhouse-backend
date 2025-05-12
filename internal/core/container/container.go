package container

import (
	"database/sql"
	auditLogRepo "warehouse/internal/auditlog"
	"warehouse/internal/integrations/googlesheets"
	"warehouse/internal/integrations/jira"
	"warehouse/internal/inventory/assets"
	"warehouse/internal/inventory/category"
	"warehouse/internal/inventory/items"
	"warehouse/internal/inventory/stocks"
	"warehouse/internal/inventory/transfers"
	"warehouse/internal/locations"
	"warehouse/internal/repository"
	"warehouse/internal/service_desk"
	"warehouse/internal/users"
	"warehouse/pkg/auditlog"
	"warehouse/pkg/security"
)

type Container struct {
	Repository          *repository.Repository
	AuditLog            *auditlog.Auditlog
	LoginHandler        *security.LoginHandler
	AssetHandler        *assets.ItemHandler
	StockHandler        *stocks.StockHandler
	LocationHandler     *locations.LocationHandler
	TransferHandler     *transfers.TransferHandler
	UserHandler         *users.UsersHandler
	ItemHandler         *items.ItemHandler
	GoogleSheetsHandler *googlesheets.GoogleSheetsHandler
	ItemCategoryHandler *category.ItemCategoryHandler
	JiraHandler         *jira.JiraHandler
	ServiceDeskHandler  *service_desk.Handler
}

func NewAppContainer(db *sql.DB) *Container {
	repo := repository.NewRepository(db)
	auditLogRepo := auditLogRepo.NewRepository(repo)
	assetRepo := assets.NewRepository(repo)
	userRepo := users.NewRepository(repo)
	auditLog := auditlog.NewAuditLog(auditLogRepo)
	userHandler := users.NewHandler(userRepo)
	loginHandler := security.NewLoginHandler(repo)
	assetHandler := assets.NewAssetHandler(repo, assetRepo, auditLog)
	stockRepo := stocks.NewRepository(repo)
	stockHandler := stocks.NewStockHandler(repo, stockRepo, auditLog)
	itemCategoryHandler := category.NewItemCategoryHandler(repo, assetRepo, stockRepo, auditLog)
	locationRepository := locations.NewLocationRepository(repo)
	locationHandler := locations.NewLocationHandler(locationRepository)
	transferRepository := transfers.NewRepository(repo)
	transferHandler := transfers.NewHandler(repo, transferRepository, assetRepo, auditLog)
	itemsHandler := items.NewItemHandler(repo, stockRepo, assetRepo, auditLogRepo)
	serviceDeskHandler := service_desk.NewHandler(repo)

	// Inicjalizacja handlera Google Sheets
	googleSheetsHandler, err := googlesheets.NewGoogleSheetsHandler()
	if err != nil {
		googleSheetsHandler = nil
	}

	// Inicjalizacja handlera Jira
	jiraHandler, err := jira.NewJiraHandler()
	if err != nil {
		jiraHandler = nil
	}

	return &Container{
		Repository:          repo,
		AuditLog:            auditLog,
		LoginHandler:        loginHandler,
		AssetHandler:        assetHandler,
		StockHandler:        stockHandler,
		LocationHandler:     locationHandler,
		TransferHandler:     transferHandler,
		UserHandler:         userHandler,
		ItemHandler:         itemsHandler,
		GoogleSheetsHandler: googleSheetsHandler,
		ItemCategoryHandler: itemCategoryHandler,
		JiraHandler:         jiraHandler,
		ServiceDeskHandler:  serviceDeskHandler,
	}
}
