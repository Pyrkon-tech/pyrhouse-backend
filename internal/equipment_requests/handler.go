package equipment_requests

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// RegisterRoutes registers equipment request routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	equipmentRoutes := router.Group("/equipment-requests")
	{
		// Sync operations
		equipmentRoutes.POST("/sync", h.ManualSync)
		equipmentRoutes.GET("/sync-log", h.GetSyncLog)

		// Quest management
		equipmentRoutes.GET("/quests", h.ListQuests)
		equipmentRoutes.GET("/quests/:id", h.GetQuest)
		equipmentRoutes.PATCH("/quests/:id/status", h.UpdateQuestStatus)

		// Transfer creation (future implementation)
		// equipmentRoutes.POST("/quests/:id/transfer", h.CreateTransferFromQuest)

		// Category mapping management
		equipmentRoutes.POST("/category-mapping", h.CreateCategoryMapping)
	}
}

// ManualSync triggers manual sync from Google Sheets and persists to database
func (h *Handler) ManualSync(c *gin.Context) {
	result, err := h.service.SyncQuestsToDatabase(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to sync equipment requests",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Sync completed successfully",
		"stats": gin.H{
			"quests_created":   result.Stats.Created,
			"quests_updated":   result.Stats.Updated,
			"quests_unchanged": result.Stats.Unchanged,
			"items_added":      result.Stats.ItemsAdded,
			"items_removed":    result.Stats.ItemsRemoved,
		},
		"quests": result.Quests,
	})
}

// ListQuests returns quests from database with filtering and pagination
func (h *Handler) ListQuests(c *gin.Context) {
	// Build filter from query params
	filter := QuestFilter{
		Status: c.Query("status"),
		Limit:  getIntQuery(c, "limit", 100),
		Offset: getIntQuery(c, "offset", 0),
	}

	// Validate limit
	if filter.Limit > 500 {
		filter.Limit = 500
	}

	quests, err := h.service.questRepo.ListQuests(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch quests",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":  len(quests),
		"limit":  filter.Limit,
		"offset": filter.Offset,
		"quests": quests,
	})
}

// GetQuest returns single quest by ID from database
func (h *Handler) GetQuest(c *gin.Context) {
	questID := c.Param("id")

	quest, err := h.service.questRepo.GetQuestByID(c.Request.Context(), questID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Quest not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, quest)
}

// UpdateQuestStatus updates quest status
func (h *Handler) UpdateQuestStatus(c *gin.Context) {
	questID := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Validate status
	validStatuses := []string{"pending", "in_progress", "completed", "cancelled"}
	if !contains(validStatuses, req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid status",
			"details": "Status must be one of: pending, in_progress, completed, cancelled",
		})
		return
	}

	err := h.service.questRepo.UpdateQuestStatus(c.Request.Context(), questID, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update quest status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Quest status updated successfully",
		"status":  req.Status,
	})
}

// GetSyncLog returns the most recent sync log
func (h *Handler) GetSyncLog(c *gin.Context) {
	log, err := h.service.questRepo.GetLatestSyncLog(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "No sync log found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, log)
}

// CreateCategoryMapping creates a manual category mapping
func (h *Handler) CreateCategoryMapping(c *gin.Context) {
	var req struct {
		FormItemName string `json:"form_item_name" binding:"required"`
		CategoryID   int    `json:"category_id" binding:"required"`
		CreatedBy    *int   `json:"created_by"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	mapping := &CategoryMapping{
		FormItemName: req.FormItemName,
		CategoryID:   req.CategoryID,
		CreatedBy:    req.CreatedBy,
	}

	err := h.service.questRepo.CreateCategoryMapping(c.Request.Context(), mapping)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create category mapping",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Category mapping created successfully",
		"mapping": mapping,
	})
}

// Helper functions

func getIntQuery(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
