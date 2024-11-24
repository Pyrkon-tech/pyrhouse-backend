package transfers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"warehouse/internal/repository"
	transfer_request "warehouse/internal/transfers/request"
	"warehouse/pkg/auditlog"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
)

type TransferHandler struct {
	Repository *repository.Repository
	AuditLog   *auditlog.Auditlog
}

func RegisterRoutes(router *gin.Engine, r *repository.Repository, a *auditlog.Auditlog) {
	handler := TransferHandler{
		Repository: r,
		AuditLog:   a,
	}

	router.GET("/transfers/:id", handler.GetTransfer)
	router.POST("/transfers", handler.CreateTransfer)
	router.PATCH("/transfers/:id/confirm", handler.UpdateTransfer)
	router.PATCH("/transfers/:id/assets/:item_id/restore-to-location", handler.RemoveAssetFromTransfer)
	router.PATCH("/transfers/:id/categories/:category_id/restore-to-location", handler.RemoveStockItemFromTransfer)
}

func (h *TransferHandler) GetTransfer(c *gin.Context) {
	transferID := c.Param("id")

	if transferID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transfer ID is required"})
		return
	}

	transfer, err := h.Repository.GetTransfer(transferID)
	if err != nil {
		log.Println("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Unable to get transfer", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transfer)
}

func (h *TransferHandler) CreateTransfer(c *gin.Context) {
	var transferRequest models.TransferRequest

	if err := c.ShouldBindJSON(&transferRequest); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		log.Fatal(err)
		return
	}
	itemTransitStatus := "in_transit"

	if len(transferRequest.SerialziedItemCollection) == 0 && len(transferRequest.UnserializedItemCollection) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Cannot create empty transport"})
		return
	}

	var err error

	validationErrors, err := h.ValidateStock(transferRequest)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Unable to verify stock"})
		log.Fatal(err)
		return
	}

	if len(validationErrors) > 0 {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Stock validation failed", "reasons": validationErrors})
		return
	}

	transferID, err := h.Repository.PerformTransfer(transferRequest, itemTransitStatus)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Unable to transfer Serialized Items", "path": "serialized_item_collection"})
		return
	}
	log.Println("transfer ID: ", transferID)
	transferRequest.TransferID = transferID

	go h.createTransferAuditLogEntry("in_transfer", transferRequest)

	c.JSON(http.StatusCreated, transferRequest)
}

func (h *TransferHandler) createTransferAuditLogEntry(action string, req models.TransferRequest) {
	// TODO handle Transfer model itself

	for _, assetID := range req.SerialziedItemCollection {
		asset := models.Asset{ID: assetID}
		go h.AuditLog.Log(
			action,
			map[string]interface{}{
				"tranfer_id":       req.TransferID,
				"from_location_id": req.FromLocationID,
				"to_location_id":   req.LocationID,
				"msg":              "Item moved in transfer",
			},
			&asset,
		)
	}

	// TODO BUG -> need proper transfer object FFS
	for _, s := range req.UnserializedItemCollection {
		stockItem := models.StockItem{Quantity: s.Quantity}
		stockItem.Category.ID = s.ItemCategoryID
		go h.AuditLog.Log(
			action,
			map[string]interface{}{
				"tranfer_id":       req.TransferID,
				"from_location_id": req.FromLocationID,
				"to_location_id":   req.LocationID,
				"msg":              "Item moved in transfer",
			},
			&stockItem,
		)
	}
}

func (h *TransferHandler) UpdateTransfer(c *gin.Context) {
	// TODO "in_transit" to "completed" or "confirmed"  do something with that
	transferID := c.Param("id")

	if transferID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transfer ID is required"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.Repository.ConfirmTransfer(transferID, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Transfer confirmed successfully",
		"transfer_id": transferID,
		"status":      req.Status,
	})
}

func (h *TransferHandler) RemoveAssetFromTransfer(c *gin.Context) {
	var req transfer_request.RemoveItemFromTransferRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid URI parameters", "details": err.Error()})
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Unable to map json body", "details": err.Error()})
		return
	}

	if req.LocationID == 0 {
		c.JSON(400, gin.H{"error": "Missing required json location_id"})
		return
	}

	err := h.Repository.RemoveAssetFromTransfer(req.ID, req.ItemID, req.LocationID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"transfer_id": req.ID})
}

func (h *TransferHandler) RemoveStockItemFromTransfer(c *gin.Context) {
	var req transfer_request.RemoveStockItemFromTransferRequest

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

	// Call the repository method
	if err := h.Repository.RemoveStockItemFromTransfer(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove stock item from transfer", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Stock item removed from transfer successfully"})
}

func (h *TransferHandler) ValidateStock(transferRequest models.TransferRequest) ([]struct {
	message  string
	property string
}, error) {
	validationState := []struct {
		message  string
		property string
	}{}

	if len(transferRequest.SerialziedItemCollection) > 0 {
		hasItemsOnStock, err := h.Repository.HasItemsInLocation(transferRequest.SerialziedItemCollection, transferRequest.FromLocationID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate serialized assets: %w", err)
		}
		if !hasItemsOnStock {
			validationState = append(validationState, struct {
				message  string
				property string
			}{
				message:  "Serialized assets are not present in location",
				property: "serialized_item_collection",
			})
		}
	}

	if len(transferRequest.UnserializedItemCollection) > 0 {
		hasEnoughQuantity, err := h.Repository.CanTransferNonSerializedItems(transferRequest.UnserializedItemCollection, transferRequest.FromLocationID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate non-serialized assets: %w", err)
		}

		if len(hasEnoughQuantity) != len(transferRequest.UnserializedItemCollection) {
			validationState = append(validationState, struct {
				message  string
				property string
			}{
				message:  "Non-serialized assets are not present in location",
				property: "unserialized_item_collection",
			})
		}
	}

	return validationState, nil
}
