package reservations

import (
	"net/http"
	"strconv"
	"warehouse/internal/security"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/assets/reservations", security.Authorize("user"), h.Reserve)
	r.GET("/assets/reservations", security.Authorize("user"), h.GetReservations)
	r.DELETE("/assets/reservations", security.Authorize("moderator"), h.Delete)
	r.POST("/assets/reservations/claim", security.Authorize("user"), h.Claim)
}

func (h *Handler) Reserve(c *gin.Context) {
	var req ReserveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	reservations, err := h.service.Reserve(c.Request.Context(), req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"reservations": reservations})
}

func (h *Handler) GetReservations(c *gin.Context) {
	var categoryID *int
	if raw := c.Query("category_id"); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid category_id"})
			return
		}
		categoryID = &id
	}

	status := c.DefaultQuery("status", "free")
	if status != "all" && status != "claimed" && status != "free" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "status must be one of: all, claimed, free"})
		return
	}

	result, err := h.service.GetReservations(c.Request.Context(), categoryID, status)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reservations", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) Delete(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}
	if len(req.PyrCodes) == 0 && len(req.IDs) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Provide pyr_codes or ids"})
		return
	}

	deleted, err := h.service.Delete(c.Request.Context(), req.PyrCodes, req.IDs)
	if err != nil {
		if claimedErr, ok := err.(*ClaimedError); ok {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"error":             "Cannot delete claimed reservations",
				"claimed_pyr_codes": claimedErr.PyrCodes,
			})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}

func (h *Handler) Claim(c *gin.Context) {
	var req ClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	createdAssets, err := h.service.Claim(c.Request.Context(), req)
	if err != nil {
		if validErr, ok := err.(*ValidationError); ok {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error":   "Claim failed: invalid pyr_codes",
				"details": validErr.Errors,
			})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to claim assets", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"created": createdAssets})
}
