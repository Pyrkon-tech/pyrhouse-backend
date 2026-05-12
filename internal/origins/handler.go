package origins

import (
	"net/http"
	"strconv"
	"strings"
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
	router.GET("/origins", h.ListActive)
	router.GET("/origins/all", security.Authorize("moderator"), h.ListAll)
	router.POST("/origins", security.Authorize("admin"), h.Create)
	router.PATCH("/origins/:id", security.Authorize("admin"), h.Update)
	router.DELETE("/origins/:id", security.Authorize("admin"), h.Delete)
}

func (h *Handler) ListActive(c *gin.Context) {
	origins, err := h.service.repo.GetAll(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch origins", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, origins)
}

func (h *Handler) ListAll(c *gin.Context) {
	origins, err := h.service.repo.GetAllIncludingInactive(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch origins", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, origins)
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	origin := Origin{
		Slug:        strings.ToLower(strings.TrimSpace(req.Slug)),
		Label:       strings.TrimSpace(req.Label),
		AllowSuffix: req.AllowSuffix,
		SortOrder:   req.SortOrder,
	}

	err := h.service.repo.Create(c.Request.Context(), &origin)
	if err != nil {
		if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Origin with this slug already exists"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create origin", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, origin)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid origin ID"})
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	updated, err := h.service.repo.Update(c.Request.Context(), id, req)
	if err != nil {
		if strings.Contains(err.Error(), "no fields") {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Origin not found"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to update origin", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid origin ID"})
		return
	}

	origin, err := h.service.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch origin", "details": err.Error()})
		return
	}
	if origin == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Origin not found"})
		return
	}

	hasEquipment, err := h.service.repo.HasAssignedEquipment(c.Request.Context(), id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to check assigned equipment", "details": err.Error()})
		return
	}
	if hasEquipment {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Cannot delete origin with assigned equipment"})
		return
	}

	if err = h.service.repo.HardDelete(c.Request.Context(), id); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete origin", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Origin deleted successfully"})
}
