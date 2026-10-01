package equipment_requests

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"warehouse/internal/security"
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
	eq := router.Group("/equipment-requests")

	// Read-only — all authenticated users
	eq.GET("/quests", h.ListQuests)
	eq.GET("/quests/unresolved-locations", h.ListUnresolvedLocationQuests)
	eq.GET("/quests/counts", h.CountQuests)
	eq.GET("/quests/:id", h.GetQuest)
	eq.GET("/quests/:id/transfer-preview", h.PreviewTransferFromQuest)
	eq.GET("/stream", h.StreamQuests)

	// Write operations — dispatcher and above
	dispatch := eq.Group("", security.Authorize("dispatcher"))
	dispatch.PATCH("/quests/:id/status", h.UpdateQuestStatus)
	dispatch.PATCH("/quests/:id/location", h.UpdateQuestLocation)
	dispatch.POST("/quests/:id/transfer", h.CreateTransferFromQuest)
}

// ListQuests returns quests from database with filtering and pagination
func (h *Handler) ListQuests(c *gin.Context) {
	filter := QuestFilter{
		Status: c.Query("status"),
		Limit:  getIntQuery(c, "limit", 100),
		Offset: getIntQuery(c, "offset", 0),
	}

	if locID := c.Query("location_id"); locID != "" {
		if id, err := strconv.Atoi(locID); err == nil {
			filter.LocationID = &id
		}
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
		"quests": nonNilQuests(quests),
	})
}

// CountQuests returns how many quests are in each status — the menu and dashboard counters.
func (h *Handler) CountQuests(c *gin.Context) {
	counts, err := h.service.questRepo.CountQuestsByStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count quests", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, counts)
}

// GetQuest returns single quest by ID from database
func (h *Handler) GetQuest(c *gin.Context) {
	questID := c.Param("id")

	quest, err := h.service.questRepo.GetQuestByID(c.Request.Context(), questID)
	if err != nil {
		if strings.Contains(err.Error(), "quest not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Quest not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quest", "details": err.Error()})
		}
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

	if quest.HasActiveTransfer() {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Quest has active transfers",
			"details": "Quest has in-progress transfers. Use transfer endpoints to change status.",
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
		case strings.Contains(errMsg, "cannot create transfer for quest with status"):
			status = http.StatusConflict
		case strings.Contains(errMsg, "could not resolve"),
			strings.Contains(errMsg, "no stock items"),
			strings.Contains(errMsg, "must include at least one stock item or asset"):
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

// StreamQuests opens an SSE connection that receives quest-related events (e.g. stock changes).
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
		"quests": nonNilQuests(quests),
	})
}

// UpdateQuestLocation manually assigns a location to a quest.
func (h *Handler) UpdateQuestLocation(c *gin.Context) {
	questID := c.Param("id")

	var req struct {
		LocationID int `json:"location_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	if _, err := h.service.questRepo.GetQuestByID(c.Request.Context(), questID); err != nil {
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

	c.JSON(http.StatusOK, gin.H{
		"message":     "Quest location updated successfully",
		"location_id": req.LocationID,
	})
}

// Helper functions

// nonNilQuests keeps an empty result a JSON array (Go encodes a nil slice as null).
func nonNilQuests(q []Quest) []Quest {
	if q == nil {
		return []Quest{}
	}
	return q
}

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
