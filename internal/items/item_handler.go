package items

import (
	"net/http"
	"warehouse/internal/repository"
	"warehouse/internal/stocks"

	"github.com/gin-gonic/gin"
)

type ItemHandler struct {
	service *ItemService
}

func NewItemHandler(r *repository.Repository, sr *stocks.StockRepository) *ItemHandler {
	return &ItemHandler{
		service: &ItemService{
			r:  r,
			sr: sr,
		},
	}
}

func (h *ItemHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/items", h.GetItems)
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
