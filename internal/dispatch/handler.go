package dispatch

import (
	"net/http"
	"strings"
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
	router.GET("/dispatch/volunteers", security.Authorize("user"), h.listVolunteers)
}

// listVolunteers returns active users with real-time dispatch status.
// Optional query param: ?status=available,on_mission
func (h *Handler) listVolunteers(c *gin.Context) {
	var statusFilter []string
	if raw := c.Query("status"); raw != "" {
		for _, s := range strings.Split(raw, ",") {
			if s = strings.TrimSpace(s); s != "" {
				statusFilter = append(statusFilter, s)
			}
		}
	}

	volunteers, err := h.repo.GetVolunteers(statusFilter)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch volunteers", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, volunteers)
}
