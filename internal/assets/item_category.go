package assets

import (
	"log"
	"net/http"
	"strconv"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
)

func (h *ItemHandler) GetItemCategories(c *gin.Context) {
	itemCategories, err := h.Repository.GetCategories()

	if err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create asset"})
		return
	}

	c.JSON(http.StatusOK, itemCategories)
}

func (h *ItemHandler) CreateItemCategory(c *gin.Context) {
	var itemCategory models.ItemCategory

	if err := c.ShouldBindJSON(&itemCategory); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	asset, err := h.Repository.PersistItemCategory(itemCategory)

	if err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create asset"})
		return
	}
	c.JSON(http.StatusCreated, asset)
}

func (h *ItemHandler) RemoveItemCategory(c *gin.Context) {
	CategoryID := c.Param("id")

	if _, err := strconv.Atoi(CategoryID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id parameter, must be an integer"})
		return
	}

	hasRelatedItems := h.Repository.HasRelatedItems(CategoryID)

	if hasRelatedItems {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "cannot delete asset category with id " + CategoryID + ": related assets exist"})
		return
	}

	err := h.Repository.DeleteItemCategoryByID(CategoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete asset category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item category deleted successfully"})
}
