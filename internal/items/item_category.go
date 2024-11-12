package items

import (
	"log"
	"net/http"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
)

func (h *ItemHandler) GetItemCategories(c *gin.Context) {
	rows, err := h.DB.Query("SELECT id, item_category, label FROM item_category")
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
