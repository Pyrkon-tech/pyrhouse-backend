package scheduling

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
	router.POST("/schedule", security.Authorize("moderator"), h.createSchedule)
	router.GET("/schedule", security.Authorize("user"), h.getSchedule)
	router.POST("/schedule/volunteers", security.Authorize("moderator"), h.importVolunteers)
	router.GET("/schedule/volunteers", security.Authorize("user"), h.getVolunteers)
	router.PATCH("/schedule/volunteers/:vid", security.Authorize("moderator"), h.updateVolunteer)
	router.POST("/schedule/generate", security.Authorize("moderator"), h.generate)
	router.DELETE("/schedule/assignments/:aid", security.Authorize("moderator"), h.deleteAssignment)
	router.POST("/schedule/assignments/swap", security.Authorize("moderator"), h.swapAssignments)
	router.GET("/schedule/validate", security.Authorize("user"), h.validate)
	router.PATCH("/schedule/publish", security.Authorize("admin"), h.publish)
	router.GET("/schedule/export", security.Authorize("moderator"), h.export)
	router.POST("/schedule/export/sheets", security.Authorize("moderator"), h.exportToSheets)
}

func (h *Handler) createSchedule(c *gin.Context) {
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	schedule, err := h.service.CreateSchedule(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create schedule", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, schedule)
}

func (h *Handler) getSchedule(c *gin.Context) {
	detail, err := h.service.GetScheduleDetail()
	if err != nil {
		if err.Error() == "no active schedule" {
			c.JSON(http.StatusNotFound, gin.H{"error": "No active schedule"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get schedule", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, detail)
}

func (h *Handler) importVolunteers(c *gin.Context) {
	var req ImportVolunteersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	if err := h.service.ImportVolunteers(req.Volunteers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import volunteers", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"imported": len(req.Volunteers)})
}

func (h *Handler) getVolunteers(c *gin.Context) {
	volunteers, err := h.service.GetVolunteers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get volunteers", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, volunteers)
}

func (h *Handler) generate(c *gin.Context) {
	detail, err := h.service.Generate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate schedule", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, detail)
}

func (h *Handler) updateVolunteer(c *gin.Context) {
	vid, err := strconv.Atoi(c.Param("vid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid volunteer ID"})
		return
	}

	var req UpdateVolunteerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	volunteer, err := h.service.UpdateVolunteer(vid, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update volunteer", "details": err.Error()})
		return
	}
	if volunteer == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Volunteer not found"})
		return
	}

	c.JSON(http.StatusOK, volunteer)
}

func (h *Handler) deleteAssignment(c *gin.Context) {
	aid, err := strconv.Atoi(c.Param("aid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment ID"})
		return
	}

	if err := h.service.DeleteAssignment(aid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete assignment", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": aid})
}

func (h *Handler) swapAssignments(c *gin.Context) {
	var req SwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	if err := h.service.SwapAssignments(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to swap assignments", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"swapped": true})
}

func (h *Handler) validate(c *gin.Context) {
	result, err := h.service.ValidateSchedule()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) publish(c *gin.Context) {
	if err := h.service.PublishSchedule(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish schedule", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "published"})
}

func (h *Handler) export(c *gin.Context) {
	csv, schedule, err := h.service.ExportCSV()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export", "details": err.Error()})
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=\"grafik-"+schedule.Name+".csv\"")
	c.String(http.StatusOK, csv)
}

func (h *Handler) exportToSheets(c *gin.Context) {
	rowsWritten, err := h.service.ExportToSheets()
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "Google Sheets integration not available" {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": "Failed to export to Google Sheets", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rows_written": rowsWritten})
}
