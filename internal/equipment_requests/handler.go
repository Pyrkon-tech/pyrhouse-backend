package equipment_requests

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	quests  []Quest // in-memory cache for Phase 1
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
		quests:  []Quest{},
	}
}

// RegisterRoutes registers equipment request routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	equipmentRoutes := router.Group("/api/equipment-requests")
	{
		equipmentRoutes.POST("/sync", h.ManualSync)
		equipmentRoutes.GET("/quests", h.ListQuests)
		equipmentRoutes.GET("/quests/:id", h.GetQuest)
	}
}

// ManualSync triggers manual sync and returns quests
func (h *Handler) ManualSync(c *gin.Context) {
	quests, err := h.service.SyncQuests(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to sync equipment requests",
			"details": err.Error(),
		})
		return
	}

	// Update in-memory cache
	h.quests = quests

	c.JSON(http.StatusOK, gin.H{
		"message": "Sync completed successfully",
		"count":   len(quests),
		"quests":  quests,
	})
}

// ListQuests returns cached quests
func (h *Handler) ListQuests(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"count":  len(h.quests),
		"quests": h.quests,
	})
}

// GetQuest returns single quest by ID
func (h *Handler) GetQuest(c *gin.Context) {
	questID := c.Param("id")

	for _, q := range h.quests {
		if q.ID == questID {
			c.JSON(http.StatusOK, q)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Quest not found"})
}
