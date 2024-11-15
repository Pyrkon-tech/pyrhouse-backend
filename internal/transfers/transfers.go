package transfers

import (
	"log"
	"net/http"
	"warehouse/internal/repository"

	"github.com/gin-gonic/gin"
)

type TransferHandler struct {
	Repository *repository.Repository
}

func RegisterRoutes(router *gin.Engine, r *repository.Repository) {
	handler := TransferHandler{Repository: r}

	router.POST("/transfers", handler.CreateTransfer)
}

func (h *TransferHandler) CreateTransfer(c *gin.Context) {
	var transferRequest TransferRequest

	if err := c.ShouldBindJSON(&transferRequest); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		log.Fatal(err)
		return
	}
	itemTransitStatus := "in_transit"

	err := h.Repository.MoveSerializedItems(transferRequest.SerialziedItemCollection, transferRequest.LocationID, itemTransitStatus)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Unable to transfer Serialized Items", "path": "serialized_item_collection"})
	}

	c.JSON(http.StatusCreated, transferRequest)
}
