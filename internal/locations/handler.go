package locations

import (
	"database/sql"
	"log"
	"net/http"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
)

type LocationHandler struct {
	DB *sql.DB
}

func RegisterRoutes(router *gin.Engine, db *sql.DB) {
	handler := LocationHandler{DB: db}

	router.POST("/locations", handler.CreateLocation)
	router.GET("/locations", handler.GetLocations)
	router.GET("/locations/:id/items", handler.GetLocationItems)
}

func (h *LocationHandler) GetLocations(c *gin.Context) {
	rows, err := h.DB.Query("SELECT id, name FROM locations")
	if err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location"})
	}
	defer rows.Close()

	var locations []models.Location
	for rows.Next() {
		var location models.Location
		if err := rows.Scan(&location.ID, &location.Name); err != nil {
			log.Fatal("Error executing SQL statement: ", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location"})
		}
		locations = append(locations, location)
	}

	if err := rows.Err(); err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location"})
	}

	c.JSON(http.StatusOK, locations)
}

func (h *LocationHandler) CreateLocation(c *gin.Context) {
	var location models.Location
	if err := c.BindJSON(&location); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		log.Fatal(err)
		return
	}

	//should be in repo
	stmtString := "INSERT INTO locations (name) VALUES ($1)"
	stmt, err := h.DB.Prepare(stmtString)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	var insertedID int
	err = h.DB.QueryRow(stmtString+" RETURNING id", location.Name).Scan(&insertedID)

	if err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location"})
		return
	}
	location.ID = insertedID
	c.JSON(http.StatusCreated, location)
}

func (h *LocationHandler) GetLocationItems(c *gin.Context) {
	// TODO: FIX LISTING
	rows, err := h.DB.Query("SELECT id, item_serial, item_category_id FROM items WHERE location_id = $1", c.Param("id"))
	if err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location"})
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		if err := rows.Scan(&item.ID, &item.Serial, &item.Category.ID); err != nil {
			log.Fatal("Error executing SQL statement: ", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location"})
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location"})
	}

	c.JSON(http.StatusOK, items)
}
