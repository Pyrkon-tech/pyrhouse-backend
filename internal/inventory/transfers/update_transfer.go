package transfers

import (
	"fmt"
	"net/http"
	"strconv"
	"warehouse/internal/repository"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
	"github.com/gin-gonic/gin"
)

func (h *TransferHandler) UpdateTransfer(c *gin.Context) {
	// TODO "in_transit" to "completed" or "confirmed"  do something with that
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
	switch req.Status {
	case "completed":
		err := h.confirmTransfer(transferID)
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
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "unsupported transfer status method:"})
	}
}

func (h *TransferHandler) confirmTransfer(transferID int) error {
	var err error
	status := "completed" // Do I need this?
	// TODO get only ids?
	assets, err := h.AssetRepo.GetTransferAssets(transferID)
	assetIDs := func(assets []models.Asset) []int {
		var ids []int
		for _, asset := range assets {
			ids = append(ids, asset.ID)
		}
		return ids
	}(*assets)

	repository.WithTransaction(h.Repository.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		if err := h.AssetRepo.UpdateItemStatus(assetIDs, "delivered"); err != nil {
			return fmt.Errorf("unable to update assets err: %w", err)
		}

		err = h.TransferRepository.ConfirmTransfer(transferID, status)
		if err != nil {
			return err
		}

		return nil
	})
	h.updateTransferAuditLog("deliver", transferID, *assets)

	return nil
}

func (h *TransferHandler) updateTransferAuditLog(action string, transferID int, assets []models.Asset) {
	for _, asset := range assets {
		go h.AuditLog.Log(
			action,
			map[string]interface{}{
				"tranfer_id": transferID,
				"msg":        "Asset arrived in transfer location",
			},
			&asset,
		)
	}
}
