package stocks

import (
	"net/http"
	"warehouse/internal/repository"
	stock_request "warehouse/internal/stocks/request"
	"warehouse/pkg/auditlog"

	"github.com/gin-gonic/gin"
)

type StockHandler struct {
	Repository *repository.Repository
	AuditLog   *auditlog.Auditlog
}

func RegisterRoutes(router *gin.Engine, r *repository.Repository, a *auditlog.Auditlog) {
	handler := StockHandler{
		Repository: r,
		AuditLog:   a,
	}

	router.POST("/stocks", handler.CreateStock)
}

func (h *StockHandler) CreateStock(c *gin.Context) {
	var stockRequest stock_request.StockItemRequest

	if err := c.ShouldBindJSON(&stockRequest); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	if stockRequest.LocationID == 0 {
		stockRequest.LocationID = 1 // setting up default location if other is not provided
	}

	stockItem, err := h.Repository.PersistStockItem(stockRequest)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create stock item"})
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
