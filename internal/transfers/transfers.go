package transfers

import (
	"fmt"
	"log"
	"net/http"
	"warehouse/internal/repository"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
)

type TransferHandler struct {
	Repository *repository.Repository
}

func RegisterRoutes(router *gin.Engine, r *repository.Repository) {
	handler := TransferHandler{Repository: r}

	router.POST("/transfers", handler.CreateTransfer)
	router.PATCH("/transfers/:id/confirm", handler.UpdateTransfer)

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

	c.JSON(http.StatusCreated, transferRequest)
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
			return nil, fmt.Errorf("failed to validate serialized items: %w", err)
		}
		if !hasItemsOnStock {
			validationState = append(validationState, struct {
				message  string
				property string
			}{
				message:  "Serialized items are not present in location",
				property: "serialized_item_collection",
			})
		}
	}

	if len(transferRequest.UnserializedItemCollection) > 0 {
		hasEnoughQuantity, err := h.Repository.CanTransferNonSerializedItems(transferRequest.UnserializedItemCollection, transferRequest.FromLocationID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate non-serialized items: %w", err)
		}

		if len(hasEnoughQuantity) != len(transferRequest.UnserializedItemCollection) {
			validationState = append(validationState, struct {
				message  string
				property string
			}{
				message:  "Non-serialized items are not present in location",
				property: "unserialized_item_collection",
			})
		}
	}

	return validationState, nil
}
