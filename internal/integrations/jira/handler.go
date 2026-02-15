package jira

import (
	"net/http"
	"warehouse/internal/security"

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		return
	}

	c.JSON(http.StatusOK, issues)
}

func (h *JiraHandler) getTaskWithComments(c *gin.Context) {
	issueID := c.Param("id")
	if issueID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing task ID"})
		return
	}

	issue, err := h.JiraService.GetTaskWithComments(issueID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch task with comments"})
		return
	}

	c.JSON(http.StatusOK, issue)
}

type ChangeStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (h *JiraHandler) changeTaskStatus(c *gin.Context) {
	issueID := c.Param("id")
	if issueID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing task ID"})
		return
	}

	var req ChangeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	response, err := h.JiraService.ChangeStatus(issueID, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change task status"})
		return
	}

	c.JSON(http.StatusOK, response)
}
