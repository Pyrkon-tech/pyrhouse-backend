package budget

import (
	"fmt"
	"net/http"
	"strings"

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
	g := router.Group("/equipment-requests")
	{
		g.GET("/budget", security.Authorize("admin"), h.GetBudgetSummary)
		g.GET("/budget/persons", security.Authorize("admin"), h.GetBudgetPersons)
		g.GET("/suppliers", security.Authorize("admin"), h.ListSuppliers)
		g.GET("/prices", security.Authorize("admin"), h.ListPrices)
		g.PUT("/prices", security.Authorize("admin"), h.UpsertPrice)
		g.DELETE("/prices", security.Authorize("admin"), h.DeletePrice)
		g.POST("/prices/sync", security.Authorize("admin"), h.SyncPricesFromSheet)
	}
}

// GET /equipment-requests/budget?budget_owner=X&vat=true
func (h *Handler) GetBudgetSummary(c *gin.Context) {
	budgetOwner := strings.TrimSpace(c.Query("budget_owner"))
	vatMultiplier := 1.0
	if c.Query("vat") == "true" {
		vatMultiplier = 1.23
	}
	summary, err := h.service.GetBudgetSummary(c.Request.Context(), budgetOwner, vatMultiplier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compute budget summary", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// GET /equipment-requests/budget/persons
func (h *Handler) GetBudgetPersons(c *gin.Context) {
	persons, err := h.service.GetBudgetPersons(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch budget persons", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"persons": persons})
}

// GET /equipment-requests/suppliers
func (h *Handler) ListSuppliers(c *gin.Context) {
	suppliers, err := h.service.ListSuppliers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch suppliers", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"suppliers": suppliers})
}

// GET /equipment-requests/prices
func (h *Handler) ListPrices(c *gin.Context) {
	prices, err := h.service.ListPrices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch price list", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"prices": prices})
}

// PUT /equipment-requests/prices
func (h *Handler) UpsertPrice(c *gin.Context) {
	var req UpsertPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}
	if err := h.service.UpsertPrice(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save price", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Price saved"})
}

// DELETE /equipment-requests/prices?item_name=Laptop&supplier=Probis
func (h *Handler) DeletePrice(c *gin.Context) {
	var req DeletePriceRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_name and supplier query params are required", "details": err.Error()})
		return
	}
	if err := h.service.DeletePrice(c.Request.Context(), req.ItemName, req.Supplier); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete price", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Price deleted"})
}

// POST /equipment-requests/prices/sync
func (h *Handler) SyncPricesFromSheet(c *gin.Context) {
	if h.service.sheetReader == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Google Sheets integration not configured"})
		return
	}
	updated, err := h.service.SyncPricesFromSheet(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync prices from sheet", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Synced %d price entries from Cennik sheet", updated),
		"updated": updated,
	})
}
