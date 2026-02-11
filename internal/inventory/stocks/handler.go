package stocks

import (
	"net/http"
	"strconv"
	"warehouse/internal/auditlog"
	"warehouse/internal/models"
	"warehouse/internal/repository"
	"warehouse/internal/security"

	"github.com/gin-gonic/gin"
)

type StockHandler struct {
	Repository      *repository.Repository
	StockRepository *StockRepository
	AuditLog        *auditlog.Auditlog
	stockService    *StockService
}

func NewStockHandler(r *repository.Repository, sr *StockRepository, a *auditlog.Auditlog, ss *StockService) *StockHandler {

	return &StockHandler{
		Repository:      r,
		StockRepository: sr,
		AuditLog:        a,
		stockService:    ss,
	}
}

func (h *StockHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/stocks", security.Authorize("user"), h.CreateStock)
	router.PATCH("/stocks/:id", security.Authorize("moderator"), h.UpdateStock)
	router.GET("/stocks", security.Authorize("user"), h.GetStocks)
	router.DELETE("/stocks/:id", security.Authorize("admin"), h.DeleteStock)
}

func (h *StockHandler) CreateStock(c *gin.Context) {
	var stockRequest models.CreateStockItemRequest
	if err := c.ShouldBindJSON(&stockRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	stock, err := h.stockService.CreateStockItem(stockRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create stock item", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, stock)
}

func (h *StockHandler) UpdateStock(c *gin.Context) {
	var stockRequest models.PatchStockItemRequest
	if err := c.ShouldBindUri(&stockRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URI parameters"})
		return
	}

	if err := c.ShouldBindJSON(&stockRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	stock, err := h.stockService.UpdateStockItem(&stockRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stock item", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stock)
}

func (h *StockHandler) GetStocks(c *gin.Context) {
	var query struct {
		LocationID    *int   `form:"location_id"`
		CategoryID    *int   `form:"category_id"`
		CategoryLabel string `form:"category_label"`
	}

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	conditions := repository.NewQueryBuilder()

	if query.LocationID != nil {
		conditions.AddCondition("location_id", *query.LocationID)
	}
	if query.CategoryID != nil {
		conditions.AddCondition("category_id", *query.CategoryID)
	}
	if query.CategoryLabel != "" {
		conditions.AddCondition("category_label", query.CategoryLabel)
	}

	stockItems, err := h.stockService.GetStockItemsBy(conditions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stock items"})
		return
	}

	c.JSON(http.StatusOK, stockItems)
}

func (h *StockHandler) DeleteStock(c *gin.Context) {
	id := c.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid stock ID"})
		return
	}

	err = h.stockService.DeleteStock(idInt)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete stock", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Stock deleted successfully"})
}
