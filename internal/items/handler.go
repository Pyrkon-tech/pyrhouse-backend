package items

import (
	"database/sql"
	"log"
	"net/http"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
)

type ItemHandler struct {
	DB *sql.DB
}

type ItemRequest struct {
	ID         int    `json:"id"`
	Serial     string `json:"serial"`
	LocationId int    `json:"location_id" default:"1"`
	Status     string `json:"status"`
	CategoryId int    `json:"category_id"`
}

func RegisterRoutes(router *gin.Engine, db *sql.DB) {
	handler := ItemHandler{DB: db}

	router.POST("/items", handler.CreateItem)
	router.GET("/items", handler.GetItems)
	router.GET("/items/categories", handler.GetItemCategories)
}

func (h *ItemHandler) GetItems(c *gin.Context) {

	c.JSON(http.StatusOK, "Hello World")
}

func (h *ItemHandler) CreateItem(c *gin.Context) {

	itemRequest := ItemRequest{
		LocationId: 1,
	}
	if err := c.BindJSON(&itemRequest); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		log.Fatal(err)
		return
	}

	item, err := h.PersistItem(itemRequest)

	if err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create item"})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *ItemHandler) PersistItem(itemRequest ItemRequest) (*models.Item, error) {
	stmtString := "INSERT INTO items (item_serial, location_id, item_category_id) VALUES ($1, $2, $3)"
	stmt, err := h.DB.Prepare(stmtString)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	var item models.Item
	err = h.DB.QueryRow(
		stmtString+" RETURNING id, item_serial, location_id, item_category_id",
		itemRequest.Serial,
		itemRequest.LocationId,
		itemRequest.CategoryId,
	).Scan(&item.ID, &item.Serial, &item.Location.ID, &item.Category.ID)

	return &item, err
}
