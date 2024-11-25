package locations

import (
	"database/sql"
	"log"
	"net/http"
	"warehouse/internal/repository"
	custom_error "warehouse/pkg/errors"
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
	router.DELETE("/locations/:id", handler.RemoveLocation)
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
	var err error
	if err = c.BindJSON(&location); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		log.Fatal(err)
		return
	}

	err = h.Repository.PersistLocation(&location)
	if _, ok := err.(*custom_error.UniqueViolationError); ok {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Could not insert, location, name not unique", "details": err.Error()})
	} else if err != nil {
		log.Println("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location"})
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	c.JSON(http.StatusCreated, location)
}

func (h *LocationHandler) GetLocationItems(c *gin.Context) {
	locationEquipment, err := h.Repository.GetLocationEquipment(c.Param("id"))

	if err != nil {
		log.Println("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location", "details": err.Error()})
	}

	c.JSON(http.StatusOK, locationEquipment)
}

func (h *LocationHandler) RemoveLocation(c *gin.Context) {
	err := h.Repository.RemoveLocation(c.Param("id"))

	if _, ok := err.(*custom_error.ForeignKeyViolationError); ok {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Could not delete location", "details": err.Error()})
		return
	} else if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not delete location", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Location deleted successfully"})
}
