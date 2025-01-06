package items

import (
	"net/http"
	"warehouse/internal/auditlog"
	"warehouse/internal/repository"
	"warehouse/internal/stocks"

	"github.com/gin-gonic/gin"
)

type ItemHandler struct {
	service *ItemService
}

func NewItemHandler(r *repository.Repository, sr *stocks.StockRepository, ar *auditlog.AuditLogRepository) *ItemHandler {
	return &ItemHandler{
		service: &ItemService{
			r:  r,
			sr: sr,
			ar: ar,
		},
	}
}

func (h *ItemHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/items", h.GetItems)
	router.GET("/items/:category/:id", h.GetItem)
}

type fetchItemQuery struct {
	ID           *int   `uri:"id" binding:"required,number"`
	CategoryType string `uri:"category" binding:"required"`
}

func (h *ItemHandler) GetItem(c *gin.Context) {
	var fetchItemQuery fetchItemQuery

	if err := c.ShouldBindUri(&fetchItemQuery); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.service.fetchItem(fetchItemQuery)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to fetch item", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *ItemHandler) GetItems(c *gin.Context) {
	var fetchItemsQuery fetchItemsQuery

	if err := c.ShouldBindQuery(&fetchItemsQuery); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items, err := h.service.fetchItems(fetchItemsQuery)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve items", "details": err.Error()})
		return
	}

	if len(items) == 0 {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	c.JSON(http.StatusOK, items)
}
