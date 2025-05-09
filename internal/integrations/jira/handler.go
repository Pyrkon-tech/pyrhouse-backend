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
