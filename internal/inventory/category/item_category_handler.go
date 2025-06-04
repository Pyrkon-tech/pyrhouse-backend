package category

import (
	"net/http"
	"strconv"
	"warehouse/internal/inventory/assets"
	"warehouse/internal/inventory/stocks"
	"warehouse/internal/repository"
	"warehouse/pkg/auditlog"
	custom_error "warehouse/pkg/errors"
	"warehouse/pkg/models"
	"warehouse/pkg/security"

	"github.com/gin-gonic/gin"
)

type ItemCategoryHandler struct {
	service  *ItemCategoryService
	ar       *assets.AssetsRepository
	sr       *stocks.StockRepository
	AuditLog *auditlog.Auditlog
}

func NewItemCategoryHandler(r *repository.Repository, ar *assets.AssetsRepository, sr *stocks.StockRepository, a *auditlog.Auditlog) *ItemCategoryHandler {
	return &ItemCategoryHandler{
		service:  NewItemCategoryService(r),
		ar:       ar,
		sr:       sr,
		AuditLog: a,
	}
}

func (h *ItemCategoryHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/assets/categories", security.Authorize("user"), h.GetItemCategories)
	router.POST("/assets/categories", security.Authorize("moderator"), h.CreateItemCategory)
	router.DELETE("/assets/categories/:id", security.Authorize("moderator"), h.RemoveItemCategory)
	router.PATCH("/assets/categories/:id", security.Authorize("admin"), h.UpdateItemCategory)
}

func (h *ItemCategoryHandler) GetItemCategories(c *gin.Context) {
	itemCategories, err := h.service.GetCategories()

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się pobrać kategorii", "details": err.Error()})
		return
	}

	if len(*itemCategories) == 0 {
		c.JSON(http.StatusOK, []models.ItemCategory{})
		return
	}

	c.JSON(http.StatusOK, itemCategories)
}

func (h *ItemCategoryHandler) CreateItemCategory(c *gin.Context) {
	var req models.ItemCategory

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	itemCategory, err := h.service.CreateCategory(req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się utworzyć kategorii", "details": err.Error()})
		return
	}

	go h.AuditLog.Log(
		"create",
		map[string]interface{}{
			"category_id": itemCategory.ID,
			"msg":         "Kategoria utworzona pomyślnie",
		},
		itemCategory,
	)

	c.JSON(http.StatusCreated, itemCategory)
}

func (h *ItemCategoryHandler) RemoveItemCategory(c *gin.Context) {
	categoryID := c.Param("id")

	if _, err := strconv.Atoi(categoryID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy format ID, musi być liczbą"})
		return
	}

	hasRelatedItems := h.ar.HasRelatedItems(categoryID)
	hasRelatedItemsInStock := h.sr.HasRelatedItems(categoryID)

	if hasRelatedItems || hasRelatedItemsInStock {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Nie można usunąć kategorii #" + categoryID + ": istnieją powiązane elementy"})
		return
	}

	err := h.service.DeleteCategory(categoryID)
	if err != nil {
		if _, ok := err.(*custom_error.ForeignKeyViolationError); ok {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Nie można usunąć kategorii #" + categoryID + ": istnieją powiązane elementy"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się usunąć kategorii"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Kategoria została usunięta pomyślnie"})
}

func (h *ItemCategoryHandler) UpdateItemCategory(c *gin.Context) {
	var req models.PatchItemCategoryRequest

	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowe parametry URI", "details": err.Error()})
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowe dane żądania", "details": err.Error(), "code": "invalid_request_payload"})
		return
	}

	updates := make(map[string]interface{})
	if req.Label != nil {
		updates["label"] = *req.Label
	}
	if req.Type != nil {
		hasRelatedItems := h.ar.HasRelatedItems(strconv.Itoa(req.ID))
		if hasRelatedItems {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"error": "Nie można zmienić typu kategorii, ponieważ ma przypisane przedmioty",
			})
			return
		}

		if h.sr.HasRelatedItems(strconv.Itoa(req.ID)) {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"error": "Nie można zmienić typu kategorii, ponieważ ma przypisane przedmioty",
			})
			return
		}

		updates["category_type"] = *req.Type
	}
	if req.PyrID != nil {
		updates["pyr_id"] = *req.PyrID
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Brak pól do aktualizacji"})
		return
	}

	err := h.service.UpdateCategory(req.ID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się zaktualizować kategorii", "details": err.Error()})
		return
	}

	go h.AuditLog.Log(
		"update",
		map[string]interface{}{
			"category_id": req.ID,
			"msg":         "Kategoria zaktualizowana pomyślnie",
		},
		&models.ItemCategory{ID: req.ID},
	)

	c.JSON(http.StatusOK, gin.H{"message": "Kategoria została zaktualizowana pomyślnie"})
}
