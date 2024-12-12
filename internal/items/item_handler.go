package items

import (
	"net/http"
	"warehouse/internal/repository"
	"warehouse/internal/stocks"

	"github.com/gin-gonic/gin"
)

type ItemHandler struct {
	Repository      *repository.Repository
	StockRepository *stocks.StockRepository
}

func NewItemHandler(r *repository.Repository, sr *stocks.StockRepository) *ItemHandler {
	return &ItemHandler{
		Repository:      r,
		StockRepository: sr,
	}
}

func (h *ItemHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/items", h.GetItems)
}

func (h *ItemHandler) GetItems(c *gin.Context) {
	var combinedItems []interface{}

	assets, err := h.Repository.GetAssets()

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unable to retrieve asset items", "details": err.Error()})
		return
	}

	stocks, err := h.StockRepository.GetStockItems()

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unable to retrieve stock items", "details": err.Error()})
		return
	}

	for _, asset := range *assets {
		combinedItems = append(combinedItems, asset)
	}
	for _, stock := range *stocks {
		combinedItems = append(combinedItems, stock)
	}

	c.JSON(http.StatusOK, combinedItems)
}
