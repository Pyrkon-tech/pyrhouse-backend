package assets

import (
	"fmt"
	"net/http"
	"strconv"
	"warehouse/internal/repository"
	"warehouse/pkg/auditlog"
	custom_error "warehouse/pkg/errors"
	"warehouse/pkg/metadata"
	"warehouse/pkg/models"
	"warehouse/pkg/security"

	"github.com/gin-gonic/gin"
)

type ItemHandler struct {
	r          *AssetsRepository
	repository *repository.Repository
	AuditLog   *auditlog.Auditlog
}

func NewAssetHandler(r *repository.Repository, ar *AssetsRepository, a *auditlog.Auditlog) *ItemHandler {
	return &ItemHandler{
		r:          ar,
		repository: r,
		AuditLog:   a,
	}
}

func (h *ItemHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/assets/pyrcode/:serial", h.GetItemByPyrCode)

	// move to main when appropriate
	protectedRoutes := router.Group("")
	protectedRoutes.Use(security.JWTMiddleware())
	{
		protectedRoutes.DELETE("/assets/:id", security.Authorize("admin"), h.RemoveAsset)
		protectedRoutes.POST("/assets/categories", h.CreateItemCategory)
		protectedRoutes.POST("/assets", h.CreateAsset)
		protectedRoutes.POST("/assets/bulk", h.CreateBulkAssets)
		protectedRoutes.GET("/assets/categories", security.Authorize("admin"), h.GetItemCategories)
		protectedRoutes.DELETE("/assets/categories/:id", h.RemoveItemCategory)
	}
}

func (h *ItemHandler) GetItemByPyrCode(c *gin.Context) {
	serial := c.Param("serial")

	if serial == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to bind serial number"})
		return
	}

	asset, err := h.r.FindItemByPyrCode(serial)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get asset", "details": err.Error()})
		return
	} else if asset.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Unable to locate status with given pyr_code"})
		return
	}

	c.JSON(http.StatusOK, asset)
}

func (h *ItemHandler) CreateAsset(c *gin.Context) {

	req := models.ItemRequest{
		LocationId: 1,
		Status:     "available",
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	origin, err := metadata.NewOrigin(req.Origin)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid asset origin",
			"details": err.Error(),
		})
		return
	}
	req.Origin = origin.String()

	categoryType, err := h.repository.GetCategoryType(req.CategoryId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unable to check category type", "details": err.Error()})
		return
	}

	if categoryType != "asset" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid category type", "details": "Category must be an asset"})
		return
	}

	asset, err := h.r.PersistItem(req)

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

	pyrCode := metadata.NewPyrCode(asset.Category.PyrID, asset.ID)
	asset.PyrCode = pyrCode.GeneratePyrCode()
	go h.r.UpdatePyrCode(asset.ID, asset.PyrCode)
	go h.AuditLog.Log(
		"create",
		map[string]interface{}{
			"serial":      asset.Serial,
			"pyr_code":    asset.PyrCode,
			"location_id": asset.Location.ID,
			"msg":         "Asset created successfully",
		},
		asset,
	)

	c.JSON(http.StatusCreated, asset)
}

func (h *ItemHandler) RemoveAsset(c *gin.Context) {
	var asset models.Asset
	var err error
	asset.ID, err = strconv.Atoi(c.Param("id"))
	if asset.ID == 0 || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to bind serial number, value must be asset ID"})
		return
	}

	res, err := h.r.CanRemoveAsset(asset.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error":   "Unable to validate asset",
			"details": err.Error(),
		})
		return
	} else if !res {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{
			"message": "Asset cannot be removed",
			"details": "Asset is either moved from stock or in dissalloved status",
		})
		return
	}

	asset.Serial, err = h.r.RemoveAsset(asset.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete asset category", "details": err.Error()})
		return
	}

	go h.AuditLog.Log(
		"remove",
		map[string]interface{}{
			"serial": asset.Serial,
			"msg":    "Remove asset from warehouse",
		},
		&asset,
	)

	c.JSON(http.StatusOK, gin.H{"message": "Asset deleted successfully"})
}

func (h *ItemHandler) CreateBulkAssets(c *gin.Context) {
	var req models.BulkItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	// Set default location ID to 1 if not specified
	if req.LocationId == 0 {
		req.LocationId = 1
	}

	// Set default status to "available" if not specified
	if req.Status == "" {
		req.Status = "available"
	}

	origin, err := metadata.NewOrigin(req.Origin)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid asset origin",
			"details": err.Error(),
		})
		return
	}
	req.Origin = origin.String()

	categoryType, err := h.repository.GetCategoryType(req.CategoryId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unable to check category type", "details": err.Error()})
		return
	}

	if categoryType != "asset" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid category type", "details": "Category must be an asset"})
		return
	}

	var createdAssets []models.Asset
	var errors []string

	for _, serial := range req.Serials {
		itemReq := models.ItemRequest{
			Serial:     serial,
			LocationId: req.LocationId,
			Status:     req.Status,
			CategoryId: req.CategoryId,
			Origin:     req.Origin,
		}

		asset, err := h.r.PersistItem(itemReq)
		if err != nil {
			switch err.(type) {
			case *custom_error.UniqueViolationError:
				errors = append(errors, fmt.Sprintf("Serial number %s already registered", serial))
				continue
			default:
				errors = append(errors, fmt.Sprintf("Failed to create asset with serial %s: %v", serial, err))
				continue
			}
		}

		pyrCode := metadata.NewPyrCode(asset.Category.PyrID, asset.ID)
		asset.PyrCode = pyrCode.GeneratePyrCode()
		go h.r.UpdatePyrCode(asset.ID, asset.PyrCode)
		go h.AuditLog.Log(
			"create",
			map[string]interface{}{
				"serial":      asset.Serial,
				"pyr_code":    asset.PyrCode,
				"location_id": asset.Location.ID,
				"msg":         "Asset created successfully",
			},
			asset,
		)

		createdAssets = append(createdAssets, *asset)
	}

	response := gin.H{
		"created": createdAssets,
	}
	if len(errors) > 0 {
		response["errors"] = errors
	}

	c.JSON(http.StatusCreated, response)
}
