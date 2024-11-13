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

func (h *ItemHandler) CreateCategory(c *gin.Context) {
	var itemCategory models.ItemCategory

	if err := c.ShouldBindJSON(&itemCategory); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.PersistItemCategory(itemCategory)

	if err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create item"})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *ItemHandler) PersistItemCategory(itemCategory models.ItemCategory) (*models.ItemCategory, error) {
	stmtString := "INSERT INTO item_category (item_category, label) VALUES ($1, $2)"
	stmt, err := h.DB.Prepare(stmtString)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	err = h.DB.QueryRow(
		stmtString+" RETURNING id",
		itemCategory.Type,
		itemCategory.Label,
	).Scan(&itemCategory.ID)

	return &itemCategory, err
}
