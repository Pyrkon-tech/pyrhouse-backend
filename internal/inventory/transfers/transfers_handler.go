package transfers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
	"warehouse/internal/inventory/assets"
	inventorylog "warehouse/internal/inventory/inventory_log"
	"warehouse/internal/inventory/stocks"
	"warehouse/internal/repository"
	"warehouse/pkg/auditlog"
	"warehouse/pkg/metadata"
	"warehouse/pkg/models"
	"warehouse/pkg/security"

	"github.com/gin-gonic/gin"
)

type TransferHandler struct {
	TransferRepository TransferRepository
	Service            *TransferService
	AssetRepo          *assets.AssetsRepository
}

type DeliveryLocationRequest struct {
	DeliveryLocation struct {
		Lat       float64   `json:"lat" binding:"required"`
		Lng       float64   `json:"lng" binding:"required"`
		Timestamp time.Time `json:"timestamp" binding:"required"`
	} `json:"delivery_location" binding:"required"`
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

func (h *TransferHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/transfers/:id", h.GetTransfer)
	router.GET("/transfers", h.RetrieveTransferList)
	router.GET("/transfers/users/:user_id", h.GetTransfersByUserAndStatus)
	router.POST("/transfers", h.CreateTransfer)
	router.PATCH("/transfers/:id/confirm", h.ConfirmTransfer)
	router.PATCH("/transfers/:id/cancel", h.CancelTransfer)
	router.PATCH("/transfers/:id/assets/:item_id/restore-to-location", h.RemoveAssetFromTransfer)
	router.PATCH("/transfers/:id/categories/:category_id/restore-to-location", h.RemoveStockItemFromTransfer)
	router.PATCH("/transfers/:id/delivery-location", h.UpdateDeliveryLocation)
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

	if req.FromLocationID == req.LocationID {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Transfer from and to location cannot be the same", "code": "same_location"})
		return
	}

	if req.LocationID == 0 || req.FromLocationID == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Location ID is required", "code": "missing_location_id"})
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

func (h *TransferHandler) GetTransfersByUserAndStatus(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowe ID użytkownika"})
		return
	}

	status := c.Query("status")
	if status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status jest wymagany"})
		return
	}

	validStatuses := map[string]bool{
		"in_transit": true,
		"completed":  true,
		"cancelled":  true,
	}
	if !validStatuses[status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy status transferu"})
		return
	}

	isAllowed := security.IsOwnerOrAllowed(c, userID, "moderator")
	if !isAllowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "Brak dostępu do tego zasobu"})
		return
	}

	transfers, err := h.Service.GetTransfersByUserAndStatus(userID, status)
	if err != nil {
		log.Printf("Unable to get transfers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nie można pobrać transferów", "details": err.Error()})
		return
	}

	if len(transfers) == 0 {
		c.JSON(http.StatusOK, []models.Transfer{})
		return
	}

	c.JSON(http.StatusOK, transfers)
}

func (h *TransferHandler) ConfirmTransfer(c *gin.Context) {
	transferID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transfer ID parameter, must be an integer"})
		return
	}

	err = h.Service.ConfirmTransfer(transferID, "completed")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to confirm transfer", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Transfer confirmed successfully",
	})
}

func (h *TransferHandler) CancelTransfer(c *gin.Context) {
	transferID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transfer ID parameter, must be an integer"})
		return
	}

	transfer, err := h.Service.GetTransfer(transferID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get transfer", "details": err.Error()})
		return
	}

	if transfer.Status != string(metadata.StatusInTransit) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transfer already in status " + transfer.Status, "details": "Cannot cancel transfer with final status"})
		return
	}

	err = h.Service.CancelTransfer(transfer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to cancel transfer", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Transfer cancelled successfully",
	})
}

func (h *TransferHandler) UpdateDeliveryLocation(c *gin.Context) {
	transferID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowe ID transferu"})
		return
	}

	var req DeliveryLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowe dane"})
		return
	}

	transfer, err := h.Service.GetTransfer(transferID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nie można pobrać transferu"})
		return
	}

	if transfer.Status != "in_transit" && transfer.Status != "completed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Można zaktualizować lokalizację tylko dla transferów w trakcie lub zakończonych"})
		return
	}

	err = h.Service.UpdateDeliveryLocation(transferID, req.DeliveryLocation.Lat, req.DeliveryLocation.Lng, req.DeliveryLocation.Timestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nie można zaktualizować lokalizacji dostawy", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Lokalizacja dostawy została zaktualizowana"})
}
