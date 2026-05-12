package scheduling

import (
	"errors"
	"net/http"
	"strconv"
	"time"
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
	router.DELETE("/schedule/:id", security.Authorize("admin"), h.deleteSchedule)
	router.POST("/schedule/volunteers", security.Authorize("moderator"), h.importVolunteers)
	router.POST("/schedule/volunteers/import-sheet", security.Authorize("moderator"), h.importVolunteersFromSheet)
	router.GET("/schedule/volunteers", security.Authorize("user"), h.getVolunteers)
	router.GET("/schedule/volunteers/me", security.Authorize("user"), h.getMySchedule)
	router.PATCH("/schedule/volunteers/:vid", security.Authorize("moderator"), h.updateVolunteer)
	router.DELETE("/schedule/volunteers/:vid", security.Authorize("admin"), h.deleteVolunteer)
	router.POST("/schedule/generate", security.Authorize("admin"), h.generate)
	router.POST("/schedule/assignments", security.Authorize("moderator"), h.addAssignment)
	router.DELETE("/schedule/assignments/:aid", security.Authorize("moderator"), h.deleteAssignment)
	router.POST("/schedule/assignments/move", security.Authorize("moderator"), h.moveAssignment)
	router.POST("/schedule/assignments/swap", security.Authorize("moderator"), h.swapAssignments)
	router.POST("/schedule/slots", security.Authorize("moderator"), h.createSlot)
	router.PATCH("/schedule/slots/:sid", security.Authorize("moderator"), h.updateSlot)
	router.DELETE("/schedule/slots/:sid", security.Authorize("moderator"), h.deleteSlot)
	router.PUT("/schedule/draft", security.Authorize("moderator"), h.saveDraft)
	router.GET("/schedule/validate", security.Authorize("user"), h.validate)
	router.POST("/schedule/validate", security.Authorize("user"), h.validateDraft)
	router.GET("/schedule/export/csv", security.Authorize("moderator"), h.export)
	router.POST("/schedule/export/sheets", security.Authorize("moderator"), h.exportToSheets)
	router.GET("/schedule/on-duty", security.Authorize("user"), h.getOnDuty)
}

func (h *Handler) createSchedule(c *gin.Context) {
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "Nieprawidłowe dane żądania", err.Error()))
		return
	}

	detail, err := h.service.CreateSchedule(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResp("create_failed", "Nie udało się utworzyć harmonogramu", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, detail)
}

func (h *Handler) getSchedule(c *gin.Context) {
	detail, err := h.service.GetScheduleDetail()
	if err != nil {
		if errors.Is(err, ErrNoActiveSchedule) {
			c.JSON(http.StatusNotFound, errorResp("not_found", "Brak aktywnego harmonogramu", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResp("fetch_failed", "Nie udało się pobrać harmonogramu", err.Error()))
		return
	}

	c.JSON(http.StatusOK, detail)
}

func (h *Handler) importVolunteers(c *gin.Context) {
	var req ImportVolunteersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "Nieprawidłowe dane żądania", err.Error()))
		return
	}

	result, err := h.service.ImportVolunteers(req.Volunteers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResp("import_failed", "Nie udało się zaimportować wolontariuszy", err.Error()))
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) getVolunteers(c *gin.Context) {
	volunteers, err := h.service.GetVolunteers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResp("fetch_failed", "Nie udało się pobrać wolontariuszy", err.Error()))
		return
	}

	c.JSON(http.StatusOK, volunteers)
}

func (h *Handler) generate(c *gin.Context) {
	detail, err := h.service.Generate()
	if err != nil {
		var genErr *GenerateBlockedError
		if errors.As(err, &genErr) {
			c.JSON(http.StatusConflict, errorRespDetails("generate_blocked",
				"Nie można wygenerować harmonogramu — wolontariusze mają już przypisane ≥10 godzin.",
				gin.H{"volunteers": genErr.Volunteers}))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResp("generate_failed", "Nie udało się wygenerować harmonogramu", err.Error()))
		return
	}

	c.JSON(http.StatusOK, detail)
}

func (h *Handler) updateVolunteer(c *gin.Context) {
	vid, err := strconv.Atoi(c.Param("vid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_id", "Nieprawidłowe ID wolontariusza", nil))
		return
	}

	var req UpdateVolunteerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "Nieprawidłowe dane żądania", err.Error()))
		return
	}

	volunteer, err := h.service.UpdateVolunteer(vid, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResp("update_failed", "Nie udało się zaktualizować wolontariusza", err.Error()))
		return
	}
	if volunteer == nil {
		c.JSON(http.StatusNotFound, errorResp("not_found", "Wolontariusz nie znaleziony", nil))
		return
	}

	c.JSON(http.StatusOK, volunteer)
}

func (h *Handler) deleteSchedule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_id", "Nieprawidłowe ID harmonogramu", nil))
		return
	}

	found, err := h.service.DeleteSchedule(id)
	if err != nil {
		if errors.Is(err, ErrEventNotEnded) {
			c.JSON(http.StatusConflict, errorResp("event_not_ended", "Nie można usunąć harmonogramu przed zakończeniem eventu", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResp("delete_failed", "Nie udało się usunąć harmonogramu", err.Error()))
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, errorResp("not_found", "Harmonogram nie znaleziony", nil))
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) getMySchedule(c *gin.Context) {
	authUserID, _ := c.Get("userID")
	authIDStr, _ := authUserID.(string)
	authID, _ := strconv.Atoi(authIDStr)

	vs, err := h.service.GetMySchedule(authID)
	if err != nil {
		if errors.Is(err, ErrNoActiveSchedule) {
			c.JSON(http.StatusNotFound, errorResp("not_found", "Brak aktywnego harmonogramu", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResp("fetch_failed", "Nie udało się pobrać grafiku", err.Error()))
		return
	}
	if vs == nil {
		c.JSON(http.StatusNotFound, errorResp("not_found", "Nie jesteś przypisany do żadnego wolontariusza w aktywnym grafiku", nil))
		return
	}

	c.JSON(http.StatusOK, vs)
}

func (h *Handler) deleteVolunteer(c *gin.Context) {
	vid, err := strconv.Atoi(c.Param("vid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_id", "Nieprawidłowe ID wolontariusza", nil))
		return
	}

	found, err := h.service.DeleteVolunteer(vid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResp("delete_failed", "Nie udało się usunąć wolontariusza", err.Error()))
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, errorResp("not_found", "Wolontariusz nie znaleziony", nil))
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) addAssignment(c *gin.Context) {
	var req AddAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "Nieprawidłowe dane żądania", err.Error()))
		return
	}

	detail, err := h.service.AddAssignment(req)
	if err != nil {
		var dupErr *DuplicateAssignmentError
		if errors.As(err, &dupErr) {
			c.JSON(http.StatusConflict, errorRespDetails("already_assigned", "Wolontariusz jest już przypisany do tego slotu", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResp("add_failed", "Nie udało się dodać przypisania", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, detail)
}

func (h *Handler) deleteAssignment(c *gin.Context) {
	aid, err := strconv.Atoi(c.Param("aid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_id", "Nieprawidłowe ID przypisania", nil))
		return
	}

	if err := h.service.DeleteAssignment(aid); err != nil {
		c.JSON(http.StatusInternalServerError, errorResp("delete_failed", "Nie udało się usunąć przypisania", err.Error()))
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) moveAssignment(c *gin.Context) {
	var req MoveAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "Nieprawidłowe dane żądania", err.Error()))
		return
	}

	resp, err := h.service.MoveAssignment(req)
	if err != nil {
		var dupErr *DuplicateAssignmentError
		if errors.As(err, &dupErr) {
			c.JSON(http.StatusConflict, errorResp("already_assigned", "Wolontariusz jest już przypisany do docelowego slotu", nil))
			return
		}
		if errors.Is(err, ErrAssignmentNotFound) {
			c.JSON(http.StatusNotFound, errorResp("not_found", "Przypisanie nie znalezione", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResp("move_failed", "Nie udało się przenieść przypisania", err.Error()))
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) swapAssignments(c *gin.Context) {
	var req SwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "Nieprawidłowe dane żądania", err.Error()))
		return
	}

	resp, err := h.service.SwapAssignments(req)
	if err != nil {
		if errors.Is(err, ErrAssignmentsNotFound) {
			c.JSON(http.StatusNotFound, errorResp("not_found", "Jedno lub oba przypisania nie znalezione", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResp("swap_failed", "Nie udało się zamienić przypisań", err.Error()))
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) validate(c *gin.Context) {
	result, err := h.service.ValidateSchedule()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResp("validate_failed", "Nie udało się zwalidować harmonogramu", err.Error()))
		return
	}

	c.JSON(http.StatusOK, result)
}


func (h *Handler) export(c *gin.Context) {
	csv, schedule, err := h.service.ExportCSV()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResp("export_failed", "Nie udało się wyeksportować harmonogramu", err.Error()))
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=\"grafik-"+schedule.Name+".csv\"")
	c.String(http.StatusOK, csv)
}

func (h *Handler) importVolunteersFromSheet(c *gin.Context) {
	var req ImportFromSheetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "Nieprawidłowe dane żądania", err.Error()))
		return
	}

	result, err := h.service.ImportVolunteersFromSheet(req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrSheetsUnavailable) {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, errorResp("import_failed", "Nie udało się zaimportować wolontariuszy z arkusza", err.Error()))
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) createSlot(c *gin.Context) {
	var req CreateSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "Nieprawidłowe dane żądania", err.Error()))
		return
	}

	slot, err := h.service.CreateSlot(req)
	if err != nil {
		h.handleServiceError(c, err, "Nie udało się utworzyć slotu")
		return
	}

	c.JSON(http.StatusCreated, slot)
}

func (h *Handler) updateSlot(c *gin.Context) {
	sid, err := strconv.Atoi(c.Param("sid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_id", "Nieprawidłowe ID slotu", nil))
		return
	}

	var req UpdateSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "Nieprawidłowe dane żądania", err.Error()))
		return
	}

	slot, err := h.service.UpdateSlot(sid, req)
	if err != nil {
		h.handleServiceError(c, err, "Nie udało się zaktualizować slotu")
		return
	}

	c.JSON(http.StatusOK, slot)
}

func (h *Handler) deleteSlot(c *gin.Context) {
	sid, err := strconv.Atoi(c.Param("sid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_id", "Nieprawidłowe ID slotu", nil))
		return
	}

	if err := h.service.DeleteSlot(sid); err != nil {
		h.handleServiceError(c, err, "Nie udało się usunąć slotu")
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) saveDraft(c *gin.Context) {
	var req SaveDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "Nieprawidłowe dane żądania", err.Error()))
		return
	}

	resp, err := h.service.SaveDraft(req)
	if err != nil {
		var vcErr *VersionConflictError
		if errors.As(err, &vcErr) {
			c.JSON(http.StatusConflict, errorRespDetails("version_conflict",
				"Harmonogram został zmieniony przez innego użytkownika.",
				gin.H{"server_version": vcErr.ServerVersion, "your_version": vcErr.YourVersion}))
			return
		}
		h.handleServiceError(c, err, "Nie udało się zapisać wersji roboczej")
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) validateDraft(c *gin.Context) {
	var req SaveDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "Nieprawidłowe dane żądania", err.Error()))
		return
	}

	result, err := h.service.ValidateDraft(req)
	if err != nil {
		h.handleServiceError(c, err, "Nie udało się zwalidować wersji roboczej")
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) exportToSheets(c *gin.Context) {
	rowsWritten, err := h.service.ExportToSheets()
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrSheetsUnavailable) {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, errorResp("export_failed", "Nie udało się wyeksportować do Google Sheets", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"rows_written": rowsWritten})
}

// handleServiceError maps common service errors to HTTP responses.
func (h *Handler) handleServiceError(c *gin.Context, err error, msg string) {
	switch {
	case errors.Is(err, ErrNoActiveSchedule):
		c.JSON(http.StatusNotFound, errorResp("not_found", "Brak aktywnego harmonogramu.", nil))
	case errors.Is(err, ErrSlotNotFound):
		c.JSON(http.StatusNotFound, errorResp("not_found", "Slot nie znaleziony.", nil))
	case errors.Is(err, ErrFestivalSlot):
		c.JSON(http.StatusForbidden, errorResp("forbidden", "Nie można usunąć slotu festiwalowego.", nil))
	case errors.Is(err, ErrFestivalSlotType):
		c.JSON(http.StatusUnprocessableEntity, errorResp("invalid_operation", "Nie można zmienić typu slotu festiwalowego.", nil))
	default:
		c.JSON(http.StatusInternalServerError, errorResp("internal_error", msg, err.Error()))
	}
}

// errorResp builds a consistent error response body.
func errorResp(slug, message string, details interface{}) gin.H {
	h := gin.H{
		"error":   slug,
		"message": message,
	}
	if details != nil {
		h["details"] = details
	}
	return h
}

func errorRespDetails(slug, message string, details gin.H) gin.H {
	return gin.H{
		"error":   slug,
		"message": message,
		"details": details,
	}
}

func (h *Handler) getOnDuty(c *gin.Context) {
	var at *time.Time
	if raw := c.Query("at"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResp("invalid_param", "Nieprawidłowy format parametru 'at' (oczekiwano RFC3339)", nil))
			return
		}
		at = &t
	}

	entries, err := h.service.GetOnDuty(at)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResp("fetch_failed", "Nie udało się pobrać dyżurujących wolontariuszy", err.Error()))
		return
	}

	c.JSON(http.StatusOK, entries)
}
