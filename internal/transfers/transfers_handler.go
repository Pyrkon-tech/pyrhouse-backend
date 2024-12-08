package transfers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"warehouse/internal/repository"
	"warehouse/internal/stocks"
	"warehouse/pkg/auditlog"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
)

type TransferHandler struct {
	Repository         *repository.Repository //TODO
	TransferRepository TransferRepository
	Service            *TransferService
	AuditLog           *auditlog.Auditlog
}

func NewHandler(r *repository.Repository, tr TransferRepository, a *auditlog.Auditlog) *TransferHandler {
	stockRepo := stocks.NewRepository(r)

	return &TransferHandler{
		Repository:         r,
		TransferRepository: tr,
		Service:            &TransferService{r, tr, stockRepo},
		AuditLog:           a,
	}
}

func (h *TransferHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/transfers/:id", h.GetTransfer)
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
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Warehouse equipment validation failed", "reasons": validationErrors})
		return
	}

	transferID, err := h.Service.PerformTransfer(transferRequest, itemTransitStatus)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Unable to transfer Serialized Items", "path": "serialized_item_collection"})
		return
	}
	transfer, err := h.Service.GetTransfer(transferID)

	if err != nil {
		c.JSON(http.StatusAccepted, gin.H{"message": "Transfer created successfully but unable to generate full object now", "id": transferID})
	}

	log.Println("transfer ID: ", transferID)

	go h.createTransferAuditLogEntry("in_transfer", transfer)

	c.JSON(http.StatusCreated, transfer)
}

func (h *TransferHandler) createTransferAuditLogEntry(action string, ts *models.Transfer) {
	go h.AuditLog.Log(
		action,
		map[string]interface{}{
			"tranfer_id":       ts.ID,
			"from_location_id": ts.FromLocation.ID,
			"to_location_id":   ts.ToLocation.ID,
			"msg":              "Transfer register",
		},
		ts,
	)

	for _, asset := range ts.AssetsCollection {
		go h.AuditLog.Log(
			action,
			map[string]interface{}{
				"tranfer_id":       ts.ID,
				"from_location_id": ts.FromLocation.ID,
				"to_location_id":   ts.ToLocation.ID,
				"msg":              "Asset moved in transfer",
			},
			&asset,
		)
	}

	for _, s := range ts.StockItemsCollection {
		// stockItem.Category.ID = s.Category.ID
		go h.AuditLog.Log(
			action,
			map[string]interface{}{
				"tranfer_id":       ts.ID,
				"from_location_id": ts.FromLocation.ID,
				"to_location_id":   ts.ToLocation.ID,
				"quantity":         s.Quantity,
				"msg":              "Stock moved in transfer",
			},
			s,
		)
	}
}

func (h *TransferHandler) RemoveAssetFromTransfer(c *gin.Context) {
	var req RemoveItemFromTransferRequest
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

type ValidationError struct {
	Message  string `json:"message"`
	Property string `json:"property"`
}

func (h *TransferHandler) ValidateStock(transferRequest models.TransferRequest) ([]ValidationError, error) {
	var validationState []ValidationError

	if len(transferRequest.SerialziedItemCollection) > 0 {
		hasItemsOnStock, err := h.Repository.HasItemsInLocation(transferRequest.SerialziedItemCollection, transferRequest.FromLocationID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate serialized assets: %w", err)
		}
		if !hasItemsOnStock {
			validationState = append(validationState, ValidationError{
				Message:  "Serialized assets are not present in location",
				Property: "serialized_item_collection",
			})
		}
	}

	if len(transferRequest.UnserializedItemCollection) > 0 {
		hasEnoughQuantity, err := h.TransferRepository.CanTransferNonSerializedItems(transferRequest.UnserializedItemCollection, transferRequest.FromLocationID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate non-serialized assets: %w", err)
		}

		if len(hasEnoughQuantity) != len(transferRequest.UnserializedItemCollection) {
			validationState = append(validationState, ValidationError{
				Message:  "Non-serialized assets are not present in location",
				Property: "unserialized_item_collection",
			})
		}
	}

	return validationState, nil
}
