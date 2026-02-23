package settings

import (
	"net/http"
	"warehouse/internal/security"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/settings", security.Authorize("admin"), h.GetAllSettings)
	router.GET("/settings/:key", security.Authorize("admin"), h.GetSetting)
	router.PUT("/settings/:key", security.Authorize("admin"), h.UpdateSetting)
}

func (h *Handler) GetAllSettings(c *gin.Context) {
	prefix := c.Query("prefix")

	var settings []AppSettings
	var err error

	if prefix != "" {
		settings, err = h.repo.GetByPrefix(prefix)
	} else {
		settings, err = h.repo.GetAll()
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

func (h *Handler) GetSetting(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setting key is required"})
		return
	}

	setting, err := h.repo.GetSetting(key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve setting"})
		return
	}
	if setting == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Setting not found"})
		return
	}

	c.JSON(http.StatusOK, setting)
}

type updateSettingRequest struct {
	Value string `json:"value" binding:"required"`
}

func (h *Handler) UpdateSetting(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setting key is required"})
		return
	}

	var req updateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	existing, err := h.repo.GetSetting(key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check setting"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Setting not found"})
		return
	}

	if err := h.repo.UpdateSetting(key, req.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update setting"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Setting updated successfully"})
}
