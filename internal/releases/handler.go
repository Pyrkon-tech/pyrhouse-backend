package releases

import (
	"net/http"
	"strconv"
	"warehouse/internal/security"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	releases := router.Group("/releases")
	{
		releases.GET("/suggest", security.Authorize("dispatcher"), h.suggest)
		releases.POST("", security.Authorize("dispatcher"), h.create)
		releases.GET("", security.Authorize("user"), h.list)
		releases.GET("/:id", security.Authorize("user"), h.get)
		releases.PUT("/:id/items", security.Authorize("dispatcher"), h.updateItems)
		releases.POST("/:id/confirm", security.Authorize("moderator"), h.confirm)
		releases.DELETE("/:id", security.Authorize("moderator"), h.delete)
	}
}

func (h *Handler) suggest(c *gin.Context) {
	originIDStr := c.Query("origin_id")
	if originIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "origin_id is required"})
		return
	}
	originID, err := strconv.Atoi(originIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid origin_id"})
		return
	}

	var locationID *int
	if locStr := c.Query("location_id"); locStr != "" {
		loc, err := strconv.Atoi(locStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid location_id"})
			return
		}
		locationID = &loc
	}

	result, err := h.service.Suggest(originID, locationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get suggestions", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) create(c *gin.Context) {
	var req CreateReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	if len(req.Assets) == 0 && len(req.Stocks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one asset or stock item is required"})
		return
	}

	userIDStr, err := security.GetUserIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID"})
		return
	}
	userID, _ := strconv.Atoi(userIDStr)

	release, err := h.service.CreateRelease(req, userID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Failed to create release", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, release)
}

func (h *Handler) list(c *gin.Context) {
	var status *string
	if s := c.Query("status"); s != "" {
		status = &s
	}

	var originID *int
	if o := c.Query("origin_id"); o != "" {
		id, err := strconv.Atoi(o)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid origin_id"})
			return
		}
		originID = &id
	}

	releases, err := h.service.ListReleases(status, originID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list releases", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, releases)
}

func (h *Handler) get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid release ID"})
		return
	}

	release, err := h.service.GetReleaseDetail(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get release", "details": err.Error()})
		return
	}
	if release == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Release not found"})
		return
	}

	c.JSON(http.StatusOK, release)
}

func (h *Handler) updateItems(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid release ID"})
		return
	}

	var req UpdateItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	if len(req.Assets) == 0 && len(req.Stocks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one asset or stock item is required"})
		return
	}

	release, err := h.service.UpdateItems(id, req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Failed to update items", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, release)
}

func (h *Handler) confirm(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid release ID"})
		return
	}

	release, err := h.service.Confirm(id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Failed to confirm release", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, release)
}

func (h *Handler) delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid release ID"})
		return
	}

	if err := h.service.DeleteRelease(id); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Failed to delete release", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Release deleted successfully"})
}
