package assets

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
)

func (h *ItemHandler) GetItemCategories(c *gin.Context) {
	itemCategories, err := h.repository.GetCategories()

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create asset", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, itemCategories)
}

func (h *ItemHandler) CreateItemCategory(c *gin.Context) {
	var req models.ItemCategory

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.PyrID == "" {
		str := req.Name
		str = str[:3]
		req.PyrID = strings.ToUpper(str)
	}

	itemCategory, err := h.repository.PersistItemCategory(req)

	if err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create asset"})
		return
	}
	c.JSON(http.StatusCreated, itemCategory)
}

func (h *ItemHandler) RemoveItemCategory(c *gin.Context) {
	CategoryID := c.Param("id")

	if _, err := strconv.Atoi(CategoryID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id parameter, must be an integer"})
		return
	}

	hasRelatedItems := h.r.HasRelatedItems(CategoryID)

	if hasRelatedItems {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "cannot delete asset category with id " + CategoryID + ": related assets exist"})
		return
	}

	err := h.repository.DeleteItemCategoryByID(CategoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete asset category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item category deleted successfully"})
}
