package dispatch

import (
	"io"
	"net/http"
	"strings"
	"warehouse/internal/security"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo        *Repository
	broadcaster *Broadcaster
}

func NewHandler(repo *Repository, broadcaster *Broadcaster) *Handler {
	return &Handler{repo: repo, broadcaster: broadcaster}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/dispatch/volunteers", security.Authorize("user"), h.listVolunteers)
	router.GET("/dispatch/stream", security.Authorize("user"), h.stream)
}

// stream opens an SSE connection that receives volunteer_status_changed and
// duty_roster_changed events.
func (h *Handler) stream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Writer.WriteHeaderNow()
	c.Writer.Flush()

	ch := h.broadcaster.Subscribe()
	defer h.broadcaster.Unsubscribe(ch)

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-ch:
			if !ok {
				return false
			}
			c.SSEvent(event.Type, event)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
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
