package service_desk

import (
	"net/http"
	"strconv"
	"time"
	"warehouse/internal/rate_limiter"
	"warehouse/internal/repository"
	"warehouse/internal/security"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service     *Service
	repository  *ServiceDeskRepository
	rateLimiter *rate_limiter.RateLimiter
}

func NewHandler(repository *repository.Repository) *Handler {
	serviceDeskRepository := NewServiceDeskRepository(repository)
	service := NewService(serviceDeskRepository)
	rateLimiter := rate_limiter.NewRateLimiter(15, time.Minute)

	return &Handler{
		service:     service,
		repository:  serviceDeskRepository,
		rateLimiter: rateLimiter,
	}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	serviceDesk := router.Group("/service-desk")
	{
		serviceDesk.GET("/requests", security.Authorize("user"), h.getRequests)
		serviceDesk.GET("/requests/:id", security.Authorize("user"), h.getRequest)
		serviceDesk.GET("/requests/:id/comments", security.Authorize("user"), h.getComments)
		serviceDesk.PUT("/requests/:id/status", security.Authorize("user"), h.changeStatus)
		serviceDesk.PUT("/requests/:id/assign", security.Authorize("user"), h.assignRequest)
		serviceDesk.PUT("/requests/:id/priority", security.Authorize("user"), h.changePriority)
		serviceDesk.POST("/requests/:id/comments", security.Authorize("user"), h.addComment)
		serviceDesk.GET("/request-types", security.Authorize("user"), h.getRequestTypes)
	}
}

func (h *Handler) RegisterPublicRoutes(router *gin.Engine) {
	serviceDesk := router.Group("/service-desk")
	{
		serviceDesk.POST("/requests", h.createRequest)
	}
}

func (h *Handler) getRequests(c *gin.Context) {
	status := c.Query("status")
	limitInt, offsetInt, err := h.getQueryPaginationParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit or offset format", "details": err.Error()})
		return
	}

	requests, err := h.repository.GetRequests(status, limitInt, offsetInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching requests", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, requests)
}

func (h *Handler) getRequest(c *gin.Context) {
	id := c.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy format ID"})
		return
	}

	req, err := h.repository.GetRequest(idInt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Zgłoszenie nie znalezione", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *Handler) createRequest(c *gin.Context) {
	userID, err := security.GetUserIDFromToken(c)

	if err != nil || userID == "" {
		clientIP := c.ClientIP()
		if !h.rateLimiter.IsAllowed(clientIP) {
			remaining := h.rateLimiter.GetRemainingRequests(clientIP)
			c.Header("X-RateLimit-Limit", "15")
			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.Header("X-RateLimit-Reset", time.Now().Add(time.Minute).Format(time.RFC3339))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":     "Rate limit exceeded. Please try again later or log in.",
				"remaining": remaining,
				"reset_at":  time.Now().Add(time.Minute).Format(time.RFC3339),
			})
			return
		}
	}

	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data format"})
		return
	}

	if userID != "" {
		id, err := strconv.Atoi(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error converting user ID"})
			return
		}
		req.CreatedByID = &id
	}

	if err := h.service.CreateRequest(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating request", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, req)
}

func (h *Handler) getComments(c *gin.Context) {
	id := c.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	comments, err := h.repository.GetComments(idInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching comments", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, comments)
}

func (h *Handler) changeStatus(c *gin.Context) {
	reqID := c.Param("id")
	id, err := strconv.Atoi(reqID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data format"})
		return
	}

	if err := h.service.ChangeStatus(id, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error changing status", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status changed"})
}

func (h *Handler) assignRequest(c *gin.Context) {
	id := c.Param("id")

	reqID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	exists, err := h.repository.RequestsExists(reqID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking if request exists", "details": err.Error()})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	var req struct {
		AssignedToID int `json:"assigned_to_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data format"})
		return
	}

	if err := h.service.AssignRequest(reqID, req.AssignedToID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error assigning request", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request assigned"})
}

func (h *Handler) changePriority(c *gin.Context) {
	id := c.Param("id")
	reqID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	exists, err := h.repository.RequestsExists(reqID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking if request exists", "details": err.Error()})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	var req struct {
		Priority string `json:"priority" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data format"})
		return
	}

	if req.Priority != "low" && req.Priority != "medium" && req.Priority != "high" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid priority"})
		return
	}

	if err := h.repository.UpdateRequestPriority(reqID, req.Priority); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error changing priority", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Priority changed"})
}

func (h *Handler) addComment(c *gin.Context) {
	id := c.Param("id")
	reqID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	exists, err := h.repository.RequestsExists(reqID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking if request exists", "details": err.Error()})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data format"})
		return
	}

	userIDstring, err := security.GetUserIDFromToken(c)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user ID", "details": err.Error()})
		return
	}

	userID, err := strconv.Atoi(userIDstring)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error converting user ID", "details": err.Error()})
		return
	}

	comment, err := h.service.AddComment(reqID, req.Content, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding comment", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, comment)
}

func (h *Handler) getRequestTypes(c *gin.Context) {
	types := h.service.GetRequestTypes()
	c.JSON(http.StatusOK, types)
}

func (h *Handler) getQueryPaginationParams(c *gin.Context) (int, int, error) {

	limit := c.Query("limit")
	offset := c.Query("offset")

	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		limitInt = 20
	}

	offsetInt, err := strconv.Atoi(offset)
	if err != nil {
		offsetInt = 0
	}

	if limitInt < 1 || limitInt > 100 {
		limitInt = 20
	}

	if offsetInt < 0 {
		offsetInt = 0
	}

	return limitInt, offsetInt, nil
}
