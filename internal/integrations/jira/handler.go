package jira

import (
	"net/http"
	"warehouse/pkg/security"

	"github.com/gin-gonic/gin"
)

type JiraHandler struct {
	JiraService *JiraService
}

func NewJiraHandler() (*JiraHandler, error) {
	return &JiraHandler{
		JiraService: NewJiraService(),
	}, nil
}

func (h *JiraHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/jira/tasks", security.Authorize("user"), h.getTasks)
	router.GET("/jira/tasks/:id", security.Authorize("user"), h.getTaskWithComments)
	router.PUT("/jira/tasks/:id/status", security.Authorize("user"), h.changeTaskStatus)
}

func (h *JiraHandler) getTasks(c *gin.Context) {
	limit := c.DefaultQuery("limit", "50")
	start := c.DefaultQuery("start", "0")
	status := c.DefaultQuery("status", "")

	issues, err := h.JiraService.GetTasks(status, limit, start)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "błąd pobierania zadań"})
		return
	}

	c.JSON(200, issues)
}

func (h *JiraHandler) getTaskWithComments(c *gin.Context) {
	issueID := c.Param("id")
	if issueID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "brak ID zadania"})
		return
	}

	issue, err := h.JiraService.GetTaskWithComments(issueID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "błąd pobierania zadania z komentarzami"})
		return
	}

	c.JSON(200, issue)
}

type ChangeStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (h *JiraHandler) changeTaskStatus(c *gin.Context) {
	issueID := c.Param("id")
	if issueID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "brak ID zadania"})
		return
	}

	var req ChangeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nieprawidłowy format danych"})
		return
	}

	response, err := h.JiraService.ChangeStatus(issueID, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "błąd zmiany statusu zadania"})
		return
	}

	c.JSON(200, response)
}
