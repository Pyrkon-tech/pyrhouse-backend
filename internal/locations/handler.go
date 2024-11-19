package locations

import (
	"database/sql"
	"log"
	"net/http"
	"warehouse/internal/repository"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
)

type LocationHandler struct {
	DB         *sql.DB
	Repository *repository.Repository
}

func RegisterRoutes(router *gin.Engine, db *sql.DB, r *repository.Repository) {
	handler := LocationHandler{DB: db, Repository: r}

	router.POST("/locations", handler.CreateLocation)
	router.GET("/locations", handler.GetLocations)
	router.GET("/locations/:id/assets", handler.GetLocationItems)
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
	locationEquipment, err := h.Repository.GetLocationEquipment(c.Param("id"))

	if err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location"})
	}

	c.JSON(http.StatusOK, locationEquipment)
}
