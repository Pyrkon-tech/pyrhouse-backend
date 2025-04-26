package locations

import (
	"log"
	"net/http"
	custom_error "warehouse/pkg/errors"
	"warehouse/pkg/models"
	"warehouse/pkg/security"

	"github.com/gin-gonic/gin"
)

type LocationHandler struct {
	Repository *LocationRepository
}

func NewLocationHandler(r *LocationRepository) *LocationHandler {
	return &LocationHandler{Repository: r}
}

func (h *LocationHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/locations", security.Authorize("moderator"), h.CreateLocation)
	router.PATCH("/locations/:id", security.Authorize("moderator"), h.UpdateLocation)
	router.GET("/locations", h.GetLocations)
	router.GET("/locations/:id/assets", h.GetLocationItems)
	router.GET("/locations/:id/search", h.SearchLocationItems)
	router.GET("locations/:id", h.GetLocationDetails)
	router.DELETE("/locations/:id", security.Authorize("moderator"), h.RemoveLocation)
}

func (h *LocationHandler) GetLocations(c *gin.Context) {
	locations, err := h.Repository.GetLocations()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not list locations", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, locations)
}

func (h *LocationHandler) GetLocationDetails(c *gin.Context) {
	locationID := c.Param("id")
	location, err := h.Repository.GetLocationDetails(locationID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not get location details", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, location)
}

func (h *LocationHandler) SearchLocationItems(c *gin.Context) {
	locationID := c.Param("id")
	searchQuery := c.Query("q")

	if len(searchQuery) < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query must be at least 1 character long"})
		return
	}

	items, err := h.Repository.SearchLocationItems(locationID, searchQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Unable to search location items",
			"details": err.Error(),
		})
		return
	}

	if len(items) == 0 {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	c.JSON(http.StatusOK, items)
}

func (h *LocationHandler) UpdateLocation(c *gin.Context) {
	var req UpdateLocationRequest

	id := c.Param("id")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing location ID"})
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	if req.Details == nil && req.Name == nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload, no fields to update"})
		return
	}

	loc, err := h.Repository.UpdateLocation(id, req)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error":   "Unable to update location, critical error",
			"details": err.Error(),
		})
	}

	c.JSON(http.StatusOK, loc)
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
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Nie można usunąć lokalizacji, ponieważ ma przypisane elementy", "details": err.Error()})
		return
	} else if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not delete location", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Location deleted successfully"})
}
