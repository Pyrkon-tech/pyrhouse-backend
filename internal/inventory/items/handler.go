package items

import (
	"net/http"
	"warehouse/internal/auditlog"
	"warehouse/internal/inventory/assets"
	"warehouse/internal/inventory/stocks"
	"warehouse/internal/models"
	"warehouse/internal/repository"

	"github.com/gin-gonic/gin"
)

type ItemHandler struct {
	repository  *repository.Repository
	assetRepo   *assets.AssetsRepository
	stockRepo   *stocks.StockRepository
	auditLog    *auditlog.Auditlog
	itemService *ItemService
}

func NewItemHandler(
	r *repository.Repository,
	sr *stocks.StockRepository,
	ar *assets.AssetsRepository,
	al *auditlog.Auditlog,
) *ItemHandler {
	return &ItemHandler{
		repository:  r,
		assetRepo:   ar,
		stockRepo:   sr,
		auditLog:    al,
		itemService: NewItemService(r, sr, ar, al),
	}
}

func (h *ItemHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/items/:category/:id", h.GetItem)
	router.GET("/items", h.GetItemList)
}

func (h *ItemHandler) GetItem(c *gin.Context) {
	var itemQuery models.RetrieveItemQuery
	if err := c.ShouldBindUri(&itemQuery); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URI parameters"})
		return
	}

	item, err := h.itemService.fetchItem(itemQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get item", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *ItemHandler) GetItemList(c *gin.Context) {
	var fetchItemsQuery models.RetrieveItemListQuery
	if err := c.ShouldBindQuery(&fetchItemsQuery); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	items, err := h.itemService.fetchItemList(fetchItemsQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get items", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}
