package items

import (
	"log"
	"net/http"
	"warehouse/internal/repository"
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
		log.Fatal(err)
		return
	}

	item, err := h.Repository.PersistItem(itemRequest)

	if err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create item"})
		return
	}
	c.JSON(http.StatusCreated, item)
}
