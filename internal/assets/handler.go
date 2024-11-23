package assets

import (
	"net/http"
	"warehouse/internal/repository"
	"warehouse/pkg/auditlog"
	custom_error "warehouse/pkg/errors"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
)

type ItemHandler struct {
	Repository *repository.Repository
	AuditLog   *auditlog.Auditlog
}

func RegisterRoutes(router *gin.Engine, r *repository.Repository, a *auditlog.Auditlog) {
	handler := ItemHandler{
		Repository: r,
		AuditLog:   a,
	}

	router.POST("/assets", handler.CreateItem)
	router.GET("/assets", handler.GetItems)
	router.POST("/assets/categories", handler.CreateItemCategory)
	router.GET("/assets/categories", handler.GetItemCategories)
	router.DELETE("/assets/categories/:id", handler.RemoveItemCategory)
}

func (h *ItemHandler) GetItems(c *gin.Context) {

	c.JSON(http.StatusOK, "Hello World")
}

func (h *ItemHandler) CreateItem(c *gin.Context) {

	itemRequest := models.ItemRequest{
		LocationId: 1,
	}
	if err := c.ShouldBindJSON(&itemRequest); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	asset, err := h.Repository.PersistItem(itemRequest)

	if err != nil {
		switch err.(type) {
		case *custom_error.UniqueViolationError:
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Item serial number already registered"})
			return
		default:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create asset"})
			return
		}
	}
	go h.AuditLog.Log(
		"create",
		map[string]interface{}{
			"serial":      asset.Serial,
			"location_id": asset.Location.ID,
			"msg":         "Register asset in warehouse",
		},
		asset,
	)

	c.JSON(http.StatusCreated, asset)
}
