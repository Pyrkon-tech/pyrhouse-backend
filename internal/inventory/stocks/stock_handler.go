package stocks

import (
	"net/http"
	"strconv"
	"warehouse/internal/repository"
	"warehouse/pkg/auditlog"
	custom_error "warehouse/pkg/errors"
	"warehouse/pkg/metadata"
	"warehouse/pkg/security"

	"github.com/gin-gonic/gin"
)

type StockHandler struct {
	Repository      *repository.Repository
	StockRepository *StockRepository
	AuditLog        *auditlog.Auditlog
}

func NewStockHandler(r *repository.Repository, sr *StockRepository, a *auditlog.Auditlog) *StockHandler {

	return &StockHandler{
		Repository:      r,
		StockRepository: sr,
		AuditLog:        a,
	}
}

func (h *StockHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/stocks", security.Authorize("user"), h.CreateStock)
	router.PATCH("/stocks/:id", security.Authorize("moderator"), h.UpdateStock)
	router.GET("/stocks", security.Authorize("user"), h.GetStocks)
	router.DELETE("/stocks/:id", security.Authorize("admin"), h.DeleteStock)
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
	origin, err := metadata.NewOrigin(stockRequest.Origin)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid asset origin",
			"details": err.Error(),
		})
		return
	}
	stockRequest.Origin = origin.String()

	stockItem, err := h.StockRepository.PersistStockItem(stockRequest)

	if err != nil {
		switch err.(type) {
		case *custom_error.UniqueViolationError:
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Stock with same data already registered"})
			return
		default:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create stock"})
			return
		}
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
	var stockRequest PatchStockItemRequest

	if err := c.ShouldBindUri(&stockRequest); err != nil {
		c.JSON(400, gin.H{"error": "Invalid URI parameters", "details": err.Error()})
		return
	}

	if err := c.ShouldBindJSON(&stockRequest); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	if stockRequest.Origin != nil {
		origin, err := metadata.NewOrigin(*stockRequest.Origin)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid asset origin",
				"details": err.Error(),
			})
			return
		}
		//TODO kinda shady
		originString := origin.String()
		stockRequest.Origin = &originString
	}

	stock, err := h.StockRepository.UpdateStock(&stockRequest)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Unable to update stock", "details": err.Error()})
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

	stockItems, err := h.StockRepository.GetStockItemsBy(conditions)
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

	err = h.StockRepository.DeleteStock(idInt)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete stock", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Stock deleted successfully"})
}
