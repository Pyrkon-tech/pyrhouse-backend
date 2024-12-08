package stocks

import (
	"net/http"
	"warehouse/internal/repository"
	"warehouse/pkg/auditlog"

	"github.com/gin-gonic/gin"
)

type StockHandler struct {
	Repository      *repository.Repository
	StockRepository *StockRepository
	AuditLog        *auditlog.Auditlog
}

func NewStockHandler(r *repository.Repository, a *auditlog.Auditlog) *StockHandler {
	stockRepo := NewRepository(r)

	return &StockHandler{
		Repository:      r,
		StockRepository: stockRepo,
		AuditLog:        a,
	}
}

func (h *StockHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/stocks", h.CreateStock)
	router.PATCH("/stocks", h.UpdateStock)
	router.GET("/stocks", h.GetStocks)
}

func (h *StockHandler) CreateStock(c *gin.Context) {
	var stockRequest StockItemRequest

	if err := c.ShouldBindJSON(&stockRequest); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	if stockRequest.LocationID == 0 {
		stockRequest.LocationID = 1 // setting up default location if other is not provided
	}

	stockItem, err := h.StockRepository.PersistStockItem(stockRequest)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create stock item", "details": err.Error()})
		return
	}

	go h.AuditLog.Log(
		"create",
		map[string]interface{}{
			"quantity":    stockItem.Quantity,
			"location_id": stockItem.Location.ID,
			"msg":         "Register stock item in warehouse",
		},
		stockItem,
	)

	c.JSON(http.StatusCreated, stockItem)
}

func (h *StockHandler) UpdateStock(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented"})
}

func (h *StockHandler) GetStocks(c *gin.Context) {
	// TODO basic get, should allow rguments like location/category_id

	stockItems, err := h.StockRepository.GetStockItems()

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create stock item"})
		return
	}

	c.JSON(http.StatusOK, stockItems)
}
