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
	router.POST("/schedules", security.Authorize("moderator"), h.createSchedule)
	router.GET("/schedules", security.Authorize("user"), h.listSchedules)
	router.GET("/schedules/:id", security.Authorize("user"), h.getSchedule)
	router.POST("/schedules/:id/volunteers", security.Authorize("moderator"), h.importVolunteers)
	router.GET("/schedules/:id/volunteers", security.Authorize("user"), h.getVolunteers)
	router.POST("/schedules/:id/generate", security.Authorize("moderator"), h.generate)
	router.DELETE("/schedules/:id/assignments/:aid", security.Authorize("moderator"), h.deleteAssignment)
	router.POST("/schedules/:id/assignments/swap", security.Authorize("moderator"), h.swapAssignments)
	router.GET("/schedules/:id/validate", security.Authorize("user"), h.validate)
	router.PATCH("/schedules/:id/publish", security.Authorize("admin"), h.publish)
	router.GET("/schedules/:id/export", security.Authorize("moderator"), h.export)
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

func (h *Handler) listSchedules(c *gin.Context) {
	schedules, err := h.service.GetSchedules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list schedules", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schedules)
}

func (h *Handler) getSchedule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	detail, err := h.service.GetScheduleDetail(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get schedule", "details": err.Error()})
		return
	}
	if detail == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	c.JSON(http.StatusOK, detail)
}

func (h *Handler) importVolunteers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	var req ImportVolunteersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	if err := h.service.ImportVolunteers(id, req.Volunteers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import volunteers", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"imported": len(req.Volunteers)})
}

func (h *Handler) getVolunteers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	volunteers, err := h.service.GetVolunteers(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get volunteers", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, volunteers)
}

func (h *Handler) generate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	detail, err := h.service.Generate(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate schedule", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, detail)
}

func (h *Handler) deleteAssignment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	aid, err := strconv.Atoi(c.Param("aid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment ID"})
		return
	}

	if err := h.service.DeleteAssignment(id, aid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete assignment", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": aid})
}

func (h *Handler) swapAssignments(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	var req SwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	if err := h.service.SwapAssignments(id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to swap assignments", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"swapped": true})
}

func (h *Handler) validate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	result, err := h.service.ValidateSchedule(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) publish(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	if err := h.service.PublishSchedule(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish schedule", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "published"})
}

func (h *Handler) export(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	slots, err := h.service.GetSlots(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get slots", "details": err.Error()})
		return
	}

	volunteers, err := h.service.GetVolunteers(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get volunteers", "details": err.Error()})
		return
	}

	assignments, err := h.service.GetAssignments(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get assignments", "details": err.Error()})
		return
	}

	schedule, err := h.service.repo.GetSchedule(id)
	if err != nil || schedule == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	csv := ExportCSV(schedule, slots, volunteers, assignments)

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=\"grafik-"+schedule.Name+".csv\"")
	c.String(http.StatusOK, csv)
}
