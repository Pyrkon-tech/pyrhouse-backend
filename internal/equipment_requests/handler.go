package equipment_requests

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service   *Service
	scheduler *Scheduler // optional — nil when auto-sync is disabled
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// SetScheduler injects the scheduler after construction (avoids circular DI).
func (h *Handler) SetScheduler(s *Scheduler) {
	h.scheduler = s
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
		equipmentRoutes.GET("/quests/unresolved-locations", h.ListUnresolvedLocationQuests)
		equipmentRoutes.GET("/quests/:id", h.GetQuest)
		equipmentRoutes.PATCH("/quests/:id/status", h.UpdateQuestStatus)
		equipmentRoutes.PATCH("/quests/:id/location", h.UpdateQuestLocation)

		// Quest → Transfer integration
		equipmentRoutes.POST("/quests/:id/transfer", h.CreateTransferFromQuest)
		equipmentRoutes.GET("/quests/:id/transfer-preview", h.PreviewTransferFromQuest)

		// Category mapping management
		equipmentRoutes.POST("/category-mapping", h.CreateCategoryMapping)
		equipmentRoutes.GET("/category-mappings", h.ListCategoryMappings)
		equipmentRoutes.DELETE("/category-mappings/:id", h.DeleteCategoryMapping)

		// Location mapping management
		equipmentRoutes.GET("/location-mappings", h.ListLocationMappings)
		equipmentRoutes.POST("/location-mappings", h.CreateLocationMapping)
		equipmentRoutes.DELETE("/location-mappings/:id", h.DeleteLocationMapping)

		// Real-time updates (SSE)
		equipmentRoutes.GET("/stream", h.StreamQuests)

		// Scheduler status
		equipmentRoutes.GET("/sync-status", h.GetSyncStatus)
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
	filter := QuestFilter{
		Status: c.Query("status"),
		Limit:  getIntQuery(c, "limit", 100),
		Offset: getIntQuery(c, "offset", 0),
	}

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

// UpdateQuestStatus updates quest status (only allowed for quests without a linked transfer)
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

	// Check if quest has a linked transfer — if so, reject manual status changes
	quest, err := h.service.questRepo.GetQuestByID(c.Request.Context(), questID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Quest not found",
			"details": err.Error(),
		})
		return
	}

	if quest.TransferID != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Quest status is managed by linked transfer",
			"details": fmt.Sprintf("Quest is linked to transfer %d. Use transfer endpoints to change status.", *quest.TransferID),
		})
		return
	}

	err = h.service.questRepo.UpdateQuestStatus(c.Request.Context(), questID, req.Status)
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

// CreateTransferFromQuest creates an inventory transfer from a quest
func (h *Handler) CreateTransferFromQuest(c *gin.Context) {
	questID := c.Param("id")

	var req CreateTransferFromQuestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	transferID, err := h.service.CreateTransferFromQuest(c.Request.Context(), questID, req)
	if err != nil {
		errMsg := err.Error()
		status := http.StatusInternalServerError

		switch {
		case strings.Contains(errMsg, "already linked to transfer"),
			strings.Contains(errMsg, "status must be"):
			status = http.StatusConflict
		case strings.Contains(errMsg, "could not resolve"),
			strings.Contains(errMsg, "no stock items"):
			status = http.StatusUnprocessableEntity
		case strings.Contains(errMsg, "quest not found"):
			status = http.StatusNotFound
		}

		c.JSON(status, gin.H{
			"error":   "Failed to create transfer from quest",
			"details": errMsg,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Transfer created from quest successfully",
		"transfer_id": transferID,
		"quest_id":    questID,
	})
}

// PreviewTransferFromQuest shows what a transfer from this quest would look like
func (h *Handler) PreviewTransferFromQuest(c *gin.Context) {
	questID := c.Param("id")
	fromLocationID := getIntQuery(c, "from_location_id", 0)

	if fromLocationID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing required parameter",
			"details": "from_location_id query parameter is required",
		})
		return
	}

	preview, err := h.service.PreviewTransferFromQuest(c.Request.Context(), questID, fromLocationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Quest not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, preview)
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

// StreamQuests opens an SSE connection that receives events whenever quests are synced.
func (h *Handler) StreamQuests(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Send 200 OK + headers immediately so the client doesn't hang waiting.
	// Without this, c.Stream() blocks on the select below and headers are never
	// flushed until the first event arrives (which may never come).
	c.Writer.WriteHeaderNow()
	c.Writer.Flush()

	ch := h.service.Subscribe()
	defer h.service.Unsubscribe(ch)

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-ch:
			if !ok {
				return false
			}
			c.SSEvent("quest_update", event)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// GetSyncStatus returns the current state of the auto-sync scheduler.
func (h *Handler) GetSyncStatus(c *gin.Context) {
	if h.scheduler == nil {
		c.JSON(http.StatusOK, SyncStatus{Enabled: false})
		return
	}

	status := SyncStatus{
		Enabled:  h.scheduler.IsEnabled(),
		Interval: h.scheduler.GetInterval().String(),
	}

	if lastSync := h.scheduler.GetLastSync(); !lastSync.IsZero() {
		status.LastSync = &lastSync
		if status.Enabled {
			nextSync := lastSync.Add(h.scheduler.GetInterval())
			status.NextSync = &nextSync
		}
	}

	if lastErr := h.scheduler.GetLastError(); lastErr != nil {
		status.LastError = lastErr.Error()
	}

	c.JSON(http.StatusOK, status)
}

// ListCategoryMappings returns all manual category mappings.
func (h *Handler) ListCategoryMappings(c *gin.Context) {
	mappings, err := h.service.questRepo.ListCategoryMappings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch category mappings",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":    len(mappings),
		"mappings": mappings,
	})
}

// DeleteCategoryMapping removes a manual category mapping by ID.
func (h *Handler) DeleteCategoryMapping(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid mapping ID",
			"details": "ID must be a positive integer",
		})
		return
	}

	if err := h.service.questRepo.DeleteCategoryMapping(c.Request.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Category mapping not found",
				"details": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete category mapping",
			"details": err.Error(),
		})
		return
	}

	c.AbortWithStatus(http.StatusNoContent)
}

// ListLocationMappings returns all manual location mappings.
func (h *Handler) ListLocationMappings(c *gin.Context) {
	mappings, err := h.service.questRepo.ListLocationMappings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch location mappings",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":    len(mappings),
		"mappings": mappings,
	})
}

// CreateLocationMapping creates a manual location mapping.
func (h *Handler) CreateLocationMapping(c *gin.Context) {
	var req struct {
		Pavilion     string `json:"pavilion" binding:"required"`
		LocationName string `json:"location_name" binding:"required"`
		LocationID   int    `json:"location_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	mapping := &LocationMapping{
		Pavilion:     req.Pavilion,
		LocationName: req.LocationName,
		LocationID:   req.LocationID,
	}

	err := h.service.questRepo.CreateLocationMapping(c.Request.Context(), mapping)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create location mapping",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Location mapping created successfully",
		"mapping": mapping,
	})
}

// DeleteLocationMapping removes a manual location mapping by ID.
func (h *Handler) DeleteLocationMapping(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid mapping ID",
			"details": "ID must be a positive integer",
		})
		return
	}

	if err := h.service.questRepo.DeleteLocationMapping(c.Request.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Location mapping not found",
				"details": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete location mapping",
			"details": err.Error(),
		})
		return
	}

	c.AbortWithStatus(http.StatusNoContent)
}

// ListUnresolvedLocationQuests returns quests with location_resolved = false.
func (h *Handler) ListUnresolvedLocationQuests(c *gin.Context) {
	quests, err := h.service.questRepo.ListUnresolvedLocationQuests(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch unresolved location quests",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":  len(quests),
		"quests": quests,
	})
}

// UpdateQuestLocation manually assigns a location to a quest. Optionally saves as mapping for future use.
func (h *Handler) UpdateQuestLocation(c *gin.Context) {
	questID := c.Param("id")

	var req struct {
		LocationID  int  `json:"location_id" binding:"required"`
		SaveMapping bool `json:"save_mapping"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	quest, err := h.service.questRepo.GetQuestByID(c.Request.Context(), questID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Quest not found",
			"details": err.Error(),
		})
		return
	}

	if err := h.service.questRepo.UpdateQuestLocationResolution(c.Request.Context(), questID, &req.LocationID, true); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update quest location",
			"details": err.Error(),
		})
		return
	}

	if req.SaveMapping {
		mapping := &LocationMapping{
			Pavilion:     quest.Destination.Pavilion,
			LocationName: quest.Destination.Location,
			LocationID:   req.LocationID,
		}
		if err := h.service.questRepo.CreateLocationMapping(c.Request.Context(), mapping); err != nil {
			// Log but don't fail - location was already assigned
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Quest location updated successfully",
		"location_id": req.LocationID,
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
