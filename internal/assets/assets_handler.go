package assets

import (
	"net/http"
	"warehouse/internal/pyrcode"
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
	router.GET("/assets/serial/:serial", handler.GetItemByPyrCode)
	router.POST("/assets/categories", handler.CreateItemCategory)
	router.GET("/assets/categories", handler.GetItemCategories)
	router.DELETE("/assets/categories/:id", handler.RemoveItemCategory)
}

func (h *ItemHandler) GetItemByPyrCode(c *gin.Context) {
	serial := c.Param("serial")

	if serial == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to bind serial number"})
		return
	}

	asset, err := h.Repository.FindItemByPyrCode(serial)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get asset", "details": err.Error()})
		return
	} else if asset.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Unable to locate status with given pyr_code"})
		return
	}

	c.JSON(http.StatusOK, asset)
}

func (h *ItemHandler) CreateItem(c *gin.Context) {

	itemRequest := models.ItemRequest{
		LocationId: 1,
		Status:     "in_stock",
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

	pyrCode := pyrcode.NewPyrCode(asset)
	asset.PyrCode = pyrCode.GeneratePyrCode()
	go h.Repository.UpdatePyrCode(asset.ID, asset.PyrCode)
	go h.AuditLog.Log(
		"create",
		map[string]interface{}{
			"serial":      asset.Serial,
			"pyr_code":    asset.PyrCode,
			"location_id": asset.Location.ID,
			"msg":         "Register asset in warehouse",
		},
		asset,
	)

	c.JSON(http.StatusCreated, asset)
}
