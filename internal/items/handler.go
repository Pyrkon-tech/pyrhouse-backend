package items

import (
	"net/http"
	"warehouse/internal/repository"
	custom_error "warehouse/pkg/errors"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
)

type ItemHandler struct {
	Repository *repository.Repository
}

func RegisterRoutes(router *gin.Engine, r *repository.Repository) {
	handler := ItemHandler{Repository: r}

	router.POST("/items", handler.CreateItem)
	router.GET("/items", handler.GetItems)
	router.POST("/items/categories", handler.CreateItemCategory)
	router.GET("/items/categories", handler.GetItemCategories)
	router.DELETE("/items/categories/:id", handler.RemoveItemCategory)
}

func (h *ItemHandler) GetItems(c *gin.Context) {

	c.JSON(http.StatusOK, "Hello World")
}

func (h *ItemHandler) CreateItem(c *gin.Context) {

	itemRequest := models.ItemRequest{
		LocationId: 1,
	}
	if err := c.ShouldBindJSON(&itemRequest); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	item, err := h.Repository.PersistItem(itemRequest)

	if err != nil {
		switch err.(type) {
		case *custom_error.UniqueViolationError:
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Item serial number already registered"})
			return
		default:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create item"})
			return
		}
	}

	c.JSON(http.StatusCreated, item)
}
