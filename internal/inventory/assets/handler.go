package assets

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"warehouse/internal/auditlog"
	apperrors "warehouse/internal/errors"
	"warehouse/internal/metadata"
	"warehouse/internal/models"
	"warehouse/internal/repository"
	"warehouse/internal/security"

	"github.com/gin-gonic/gin"
)

type ItemHandler struct {
	r            *AssetsRepository
	repository   *repository.Repository
	AuditLog     *auditlog.Auditlog
	assetService *AssetService
}

func NewAssetHandler(r *repository.Repository, ar *AssetsRepository, a *auditlog.Auditlog) *ItemHandler {
	return &ItemHandler{
		r:            ar,
		repository:   r,
		AuditLog:     a,
		assetService: NewAssetService(ar, r, a),
	}
}

func (h *ItemHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/assets/pyrcode/:serial", h.GetItemByPyrCode)
	router.POST("/assets", security.Authorize("user"), h.CreateAsset)
	router.POST("/assets/bulk", security.Authorize("user"), h.CreateBulkAssets)
	router.POST("/assets/without-serial", security.Authorize("user"), h.CreateAssetWithoutSerial)
	router.DELETE("/assets/:id", security.Authorize("moderator"), h.RemoveAsset)
	router.PATCH("/assets/:id/serial", security.Authorize("moderator"), h.UpdateAssetSerial)
	router.PATCH("/assets/:id/logs/location", security.Authorize("user"), h.UpdateAssetLocation)
	router.GET("/assets/report", security.Authorize("moderator"), h.GetAssetsReport)
	router.GET("/stocks/report", security.Authorize("moderator"), h.GetStockReport)
}

func (h *ItemHandler) GetItemByPyrCode(c *gin.Context) {
	serial := c.Param("serial")

	if serial == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to bind serial number"})
		return
	}

	asset, err := h.assetService.FindByPyrCode(serial)

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

	if req.Serial == nil || *req.Serial == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Numer seryjny nie może być pusty"})
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

	asset, err := h.assetService.CreateAsset(req)
	if err != nil {
		switch err.(type) {
		case *apperrors.UniqueViolationError:
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Item serial number already registered"})
			return
		default:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create asset", "details": err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, asset)
}

func (h *ItemHandler) RemoveAsset(c *gin.Context) {
	assetID, err := strconv.Atoi(c.Param("id"))
	if assetID == 0 || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to bind serial number, value must be asset ID"})
		return
	}

	canRemove, err := h.assetService.CanRemoveAsset(assetID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error":   "Unable to validate asset",
			"details": err.Error(),
		})
		return
	}
	if !canRemove {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{
			"message": "Asset cannot be removed",
			"details": "Asset is either moved from stock or in dissalloved status",
		})
		return
	}

	_, err = h.assetService.RemoveAsset(assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete asset", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Asset deleted successfully"})
}

func (h *ItemHandler) CreateBulkAssets(c *gin.Context) {
	var req models.BulkItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy format żądania", "details": err.Error()})
		return
	}

	locationId, status, origin, err := h.getRequestDefaults(req.LocationId, req.Status, req.Origin)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Nie udało się pobrać wartości domyślnych", "details": err.Error()})
		return
	}

	req.LocationId = locationId
	req.Status = status
	req.Origin = origin

	categoryType, err := h.repository.GetCategoryType(req.CategoryId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Nie udało się sprawdzić typu kategorii", "details": err.Error()})
		return
	}

	if categoryType != "asset" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy typ kategorii", "details": "Kategoria musi być typu Sprzęt(asset)"})
		return
	}

	createdAssets, errors, err := h.assetService.CreateBulkAssets(req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Wystąpił nieoczekiwany błąd podczas tworzenia zasobów zbiorczo", "details": err.Error()})
		return
	}
	if len(errors) > 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Wystąpiły błędy podczas tworzenia zasobów zbiorczo, nie utworzono żadnych zasobów", "errors": errors})
		return
	}

	response := gin.H{
		"created": createdAssets,
	}

	c.JSON(http.StatusCreated, response)
}

func (h *ItemHandler) CreateAssetWithoutSerial(c *gin.Context) {
	var req models.EmergencyAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy format żądania", "details": err.Error()})
		return
	}

	locationId, status, origin, err := h.getRequestDefaults(req.LocationId, req.Status, req.Origin)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Nie udało się pobrać wartości domyślnych", "details": err.Error()})
		return
	}

	req.LocationId = locationId
	req.Status = status
	req.Origin = origin

	categoryType, err := h.repository.GetCategoryType(req.CategoryId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Nie udało się sprawdzić typu kategorii", "details": err.Error()})
		return
	}

	if categoryType != "asset" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy typ kategorii", "details": "Kategoria musi być typu Sprzęt(asset)"})
		return
	}

	createdAssets, errors, err := h.assetService.CreateAssetsWithoutSerial(req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się utworzyć zasobów awaryjnych", "details": err.Error()})
		return
	}

	response := gin.H{
		"created": createdAssets,
	}
	if len(errors) > 0 {
		response["errors"] = errors
	}

	c.JSON(http.StatusCreated, response)
}

func (h *ItemHandler) getRequestDefaults(locationId int, status string, origin string) (int, string, string, error) {

	if locationId == 0 {
		locationId = 1
	}

	if status == "" {
		status = "available"
	}

	o, err := metadata.NewOrigin(origin)
	if err != nil {
		return 0, "", "", err
	}
	origin = o.String()

	return locationId, status, origin, nil
}

func (h *ItemHandler) UpdateAssetSerial(c *gin.Context) {
	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy format ID zasobu"})
		return
	}

	var req struct {
		Serial string `json:"serial" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy format żądania", "details": err.Error()})
		return
	}

	updatedAsset, err := h.assetService.UpdateAssetSerial(assetID, req.Serial)
	if err != nil {
		switch err.(type) {
		case *apperrors.UniqueViolationError:
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Numer seryjny już istnieje"})
			return
		default:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się zaktualizować numeru seryjnego", "details": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, updatedAsset)
}

func (h *ItemHandler) GetAssetsReport(c *gin.Context) {
	assets, err := h.assetService.GetAssetsForReport()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się wygenerować raportu", "details": err.Error()})
		return
	}

	csvData := [][]string{
		{"ID", "Kategoria", "Numer seryjny", "Kod PYR", "Pochodzenie", "Status", "Typ kategorii", "Lokalizacja"},
	}

	for _, asset := range assets {
		csvData = append(csvData, []string{
			fmt.Sprintf("%d", asset.ID),
			asset.CategoryLabel,
			asset.Serial.String,
			asset.PyrCode.String,
			asset.Origin,
			asset.Status,
			asset.CategoryType,
			asset.LocationName,
		})
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=raport_sprzetu.csv")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Cache-Control", "no-cache")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	writer := csv.NewWriter(c.Writer)
	for _, record := range csvData {
		if err := writer.Write(record); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Błąd podczas generowania CSV", "details": err.Error()})
			return
		}
	}
	writer.Flush()
}

func (h *ItemHandler) GetStockReport(c *gin.Context) {
	stock, err := h.assetService.GetStockForReport()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się wygenerować raportu", "details": err.Error()})
		return
	}

	csvData := [][]string{
		{"ID", "Kategoria", "Pochodzenie", "Ilość", "Lokalizacja"},
	}

	for _, item := range stock {
		csvData = append(csvData, []string{
			fmt.Sprintf("%d", item.ID),
			item.CategoryLabel,
			item.Origin,
			strconv.Itoa(item.Quantity),
			item.LocationName,
		})
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=raport_magazynu.csv")
	c.Header("Cache-Control", "no-cache")

	writer := csv.NewWriter(c.Writer)
	for _, record := range csvData {
		if err := writer.Write(record); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Błąd podczas generowania CSV", "details": err.Error()})
			return
		}
	}
	writer.Flush()
}

func (h *ItemHandler) UpdateAssetLocation(c *gin.Context) {
	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy format ID zasobu"})
		return
	}
	var reqLocation models.DeliveryLocationRequest
	if err := c.ShouldBindJSON(&reqLocation); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy format żądania", "details": err.Error()})
		return
	}

	err = h.assetService.UpdateAssetLocation(assetID, reqLocation.DeliveryLocation)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się zaktualizować lokalizacji zasobu", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Lokalizacja zasobu zaktualizowana"})
}
