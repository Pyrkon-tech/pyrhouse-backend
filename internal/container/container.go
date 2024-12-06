package container

import (
	"database/sql"
	"warehouse/internal/assets"
	auditLogRepo "warehouse/internal/auditlog"
	"warehouse/internal/locations"
	"warehouse/internal/repository"
	"warehouse/internal/stocks"
	"warehouse/internal/transfers"
	"warehouse/internal/users"
	"warehouse/pkg/auditlog"
	"warehouse/pkg/security"
)

type Container struct {
	Repository      *repository.Repository
	AuditLog        *auditlog.Auditlog
	LoginHandler    *security.LoginHandler
	AssetHandler    *assets.ItemHandler
	StockHandler    *stocks.StockHandler
	LocationHandler *locations.LocationHandler
	TransferHandler *transfers.TransferHandler
	UserHandler     *users.UsersHandler
}

func NewAppContainer(db *sql.DB) *Container {
	repo := repository.NewRepository(db)
	auditLogRepo := auditLogRepo.NewRepository(repo)
	userRepo := users.NewRepository(repo)
	auditLog := auditlog.NewAuditLog(auditLogRepo)
	userHandler := users.NewHandler(userRepo)
	loginHandler := security.NewLoginHandler(repo)
	assetHandler := assets.NewAssetHandler(repo, auditLog)
	stockHandler := stocks.NewStockHandler(repo, auditLog)
	locationRepository := locations.NewLocationRepository(repo)
	locationHandler := locations.NewLocationHandler(locationRepository)

	transferHandler := transfers.NewHandler(repo, auditLog)

	return &Container{
		Repository:      repo,
		AuditLog:        auditLog,
		LoginHandler:    loginHandler,
		AssetHandler:    assetHandler,
		StockHandler:    stockHandler,
		LocationHandler: locationHandler,
		TransferHandler: transferHandler,
		UserHandler:     userHandler,
	}
}
