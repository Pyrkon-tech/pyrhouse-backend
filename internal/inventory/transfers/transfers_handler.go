package transfers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"warehouse/internal/inventory/assets"
	inventorylog "warehouse/internal/inventory/inventory_log"
	"warehouse/internal/inventory/stocks"
	"warehouse/internal/repository"
	"warehouse/pkg/auditlog"
	"warehouse/pkg/metadata"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
)

type TransferHandler struct {
	TransferRepository TransferRepository
	Service            *TransferService
	AssetRepo          *assets.AssetsRepository
}

func NewHandler(r *repository.Repository, tr TransferRepository, ar *assets.AssetsRepository, a *auditlog.Auditlog) *TransferHandler {
	stockRepo := stocks.NewRepository(r)
	inventorylog := inventorylog.NewInventoryLog(a)

	return &TransferHandler{
		TransferRepository: tr,
		Service:            &TransferService{r, tr, ar, stockRepo, inventorylog},
		AssetRepo:          ar,
	}
}

func (h *TransferHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/transfers/:id", h.GetTransfer)
	router.GET("/transfers", h.RetrieveTransferList)
	router.POST("/transfers", h.CreateTransfer)
	router.PATCH("/transfers/:id/confirm", h.UpdateTransfer)
	router.PATCH("/transfers/:id/assets/:item_id/restore-to-location", h.RemoveAssetFromTransfer)
	router.PATCH("/transfers/:id/categories/:category_id/restore-to-location", h.RemoveStockItemFromTransfer)
}

func (h *TransferHandler) GetTransfer(c *gin.Context) {
	transferID, err := strconv.Atoi(c.Param("id"))

	if err != nil || transferID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transfer ID is required"})
		return
	}

	transfer, err := h.Service.GetTransfer(transferID)
	if err != nil {
		log.Println("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Unable to get transfer", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transfer)
}

func (h *TransferHandler) RetrieveTransferList(c *gin.Context) {
	transfers, err := h.Service.GetTransfers()
	if err != nil {
		log.Println("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Unable to get transfer", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transfers)
}

func (h *TransferHandler) CreateTransfer(c *gin.Context) {
	var req models.TransferRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		log.Println("Error binding JSON:", err)
		return
	}
	itemTransitStatus := "in_transit"

	if len(req.AssetItemCollection) == 0 && len(req.StockItemCollection) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Cannot create empty transfer"})
		return
	}

	validationErrors, err := h.Service.ValidateStock(req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Unable to verify stock"})
		log.Println("Error validating stock:", err)
		return
	}

	if len(validationErrors) > 0 {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Warehouse equipment validation failed", "reasons": validationErrors})
		return
	}

	transferID, err := h.Service.InitTransfer(req, itemTransitStatus)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Unable to transfer items", "details": err.Error()})
		return
	}
	
	transfer, err := h.Service.GetTransfer(transferID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusAccepted, gin.H{"message": "Transfer created successfully but unable to generate full object now", "id": transferID, "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, transfer)
}

func (h *TransferHandler) RemoveAssetFromTransfer(c *gin.Context) {
	var req RemoveItemFromTransferRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URI parameters", "details": err.Error()})
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to map json body", "details": err.Error()})
		return
	}

	if req.LocationID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required json location_id"})
		return
	}

	err := h.AssetRepo.RemoveAssetFromTransfer(req.ID, req.ItemID, req.LocationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transfer_id": req.ID})
}

// RemoveStockItemFromTransfer obsługuje zapytanie o usunięcie pozycji magazynowej z transferu
func (h *TransferHandler) RemoveStockItemFromTransfer(c *gin.Context) {
	var req stocks.RemoveStockItemFromTransferRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to map JSON body", "details": err.Error()})
		return
	}

	getAndConvertParam := func(key string) (int, error) {
		param := c.Param(key)
		if param == "" {
			return 0, fmt.Errorf("parameter %s missing from URI", key)
		}
		value, err := strconv.Atoi(param)
		if err != nil {
			return 0, fmt.Errorf("invalid %s format: %w", key, err)
		}
		return value, nil
	}

	transferID, err := getAndConvertParam("id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	categoryID, err := getAndConvertParam("category_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.TransferID = transferID
	req.CategoryID = categoryID

	if err := h.Service.RemoveStockItemFromTransfer(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove stock item from transfer", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Stock item removed from transfer successfully"})
}

func (h *TransferHandler) UpdateTransfer(c *gin.Context) {
	transferID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transfer ID parameter, must be an integer"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status, _ := metadata.NewStatus(req.Status)

	switch status {
	case "completed":
		err := h.Service.confirmTransfer(transferID, string(status))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to update transfer status", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":     "Transfer confirmed successfully",
			"transfer_id": transferID,
			"status":      req.Status,
		})
	default:
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Unsupported transfer status method"})
	}
}
