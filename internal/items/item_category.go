package items

import (
	"log"
	"net/http"
	"strconv"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
)

func (h *ItemHandler) GetItemCategories(c *gin.Context) {
	rows, err := h.Repository.DB.Query("SELECT id, item_category, label FROM item_category")
	if err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location"})
	}
	defer rows.Close()

	var itemCategories []models.ItemCategory
	for rows.Next() {
		var itemCategory models.ItemCategory
		if err := rows.Scan(&itemCategory.ID, &itemCategory.Type, &itemCategory.Label); err != nil {
			log.Fatal("Error executing SQL statement: ", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location"})
		}
		itemCategories = append(itemCategories, itemCategory)
	}

	if err := rows.Err(); err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location"})
	}

	c.JSON(http.StatusOK, itemCategories)
}

func (h *ItemHandler) CreateItemCategory(c *gin.Context) {
	var itemCategory models.ItemCategory

	if err := c.ShouldBindJSON(&itemCategory); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.Repository.PersistItemCategory(itemCategory)

	if err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create item"})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *ItemHandler) RemoveItemCategory(c *gin.Context) {
	CategoryID := c.Param("id")

	// Validate that id is an integer
	if _, err := strconv.Atoi(CategoryID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id parameter, must be an integer"})
		return
	}

	hasRelatedItems := h.Repository.HasRelatedItems(CategoryID)

	if hasRelatedItems {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "cannot delete item category with id " + CategoryID + ": related items exist"})
		return
	}

	err := h.Repository.DeleteItemCategoryByID(CategoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete item category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item category deleted successfully"})
}
