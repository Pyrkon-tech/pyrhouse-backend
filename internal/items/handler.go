package items

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
)

type ItemHandler struct {
	DB *sql.DB
}

func RegisterRoutes(router *gin.Engine, db *sql.DB) {
	handler := ItemHandler{DB: db}

	router.POST("/items", handler.CreateItem)
	router.GET("/items", handler.GetItems)
	router.GET("/items/categories", handler.GetItemCategories)
}

func (h *ItemHandler) GetItems(c *gin.Context) {
	// var itemList []models.Item

	c.JSON(http.StatusOK, "Hello World")
}

func (h *ItemHandler) CreateItem(c *gin.Context) {

	var item models.Item
	if err := c.BindJSON(&item); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		log.Fatal(err)
		return
	}
	fmt.Println("1. " + item.Serial + " 2. " + item.Type)
	// default locations should be always 1
	item.LocationId = 1

	//should be in repo
	stmtString := "INSERT INTO items (item_type, item_serial, location_id) VALUES ($1, $2, $3)"
	stmt, err := h.DB.Prepare(stmtString)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	var insertedID int
	err = h.DB.QueryRow(stmtString+" RETURNING id", item.Type, item.Serial, item.LocationId).Scan(&insertedID)

	if err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert item"})
		return
	}
	item.ID = insertedID
	c.JSON(http.StatusCreated, item)
}
