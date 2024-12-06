package locations

import (
	"log"
	"net/http"
	custom_error "warehouse/pkg/errors"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
)

type LocationHandler struct {
	Repository *LocationRepository
}

func NewLocationHandler(r *LocationRepository) *LocationHandler {
	return &LocationHandler{Repository: r}
}

func (h *LocationHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/locations", h.CreateLocation)
	router.GET("/locations", h.GetLocations)
	router.GET("/locations/:id/assets", h.GetLocationItems)
	router.DELETE("/locations/:id", h.RemoveLocation)
}

func (h *LocationHandler) GetLocations(c *gin.Context) {
	locations, err := h.Repository.GetLocations()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not list locations", "details": err.Error()})
		return
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
		return
	} else if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location"})
		return
	}

	c.JSON(http.StatusCreated, location)
}

func (h *LocationHandler) GetLocationItems(c *gin.Context) {
	locationEquipment, err := h.Repository.GetLocationEquipment(c.Param("id"))

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not get location items", "details": err.Error()})
		return
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
