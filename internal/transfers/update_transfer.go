package transfers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"warehouse/internal/repository"

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
		err := h.confirmTransfer(transferID, req.Status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

func (h *TransferHandler) confirmTransfer(transferID int, status string) error {
	var err error
	stockItems, stockErr := h.Repository.GetStockItemsByTransfer(transferID)
	assets, err := h.Repository.GetTransferAssets(transferID)

	if stockErr != nil || err != nil {
		return fmt.Errorf("unable to obtain transfer equipment: %s", err.Error())
	}

	repository.WithTransaction(h.Repository.GoquDBWrapper, func(tx *goqu.TxDatabase) error {

		// Last step
		// err = h.Repository.ConfirmTransfer(transferID, status)
		// if err != nil {
		// 	return err
		// }

		return nil
	})

	log.Println("Stock ID", stockItems)
	log.Println("Assets ID", assets)

	return nil
}
