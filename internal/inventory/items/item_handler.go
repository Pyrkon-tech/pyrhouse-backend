package items

import (
	"net/http"
	"warehouse/internal/auditlog"
	"warehouse/internal/inventory/assets"
	"warehouse/internal/inventory/stocks"
	"warehouse/internal/repository"

	"github.com/gin-gonic/gin"
)

type ItemHandler struct {
	service *ItemService
}

func NewItemHandler(r *repository.Repository, sr *stocks.StockRepository, ar *assets.AssetsRepository, auditLogRepo *auditlog.AuditLogRepository) *ItemHandler {
	return &ItemHandler{
		service: &ItemService{
			r:                  r,
			sr:                 sr,
			ar:                 ar,
			auditlogRepository: auditLogRepo,
		},
	}
}

func (h *ItemHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/items", h.RetrieveItemList)
	router.GET("/items/:category/:id", h.RetrieveItem)
}

func (h *ItemHandler) RetrieveItem(c *gin.Context) {
	var itemQuery retrieveItemQuery

	if err := c.ShouldBindUri(&itemQuery); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.service.fetchItem(itemQuery)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to fetch item", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *ItemHandler) RetrieveItemList(c *gin.Context) {
	var fetchItemsQuery retrieveItemListQuery

	if err := c.ShouldBindQuery(&fetchItemsQuery); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items, err := h.service.fetchItemList(fetchItemsQuery)

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
