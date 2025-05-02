package category

import (
	"log"
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
	ar         *assets.AssetsRepository
	sr         *stocks.StockRepository
	repository *repository.Repository
	AuditLog   *auditlog.Auditlog
}

func NewItemCategoryHandler(r *repository.Repository, ar *assets.AssetsRepository, sr *stocks.StockRepository, a *auditlog.Auditlog) *ItemCategoryHandler {
	return &ItemCategoryHandler{
		ar:         ar,
		sr:         sr,
		repository: r,
		AuditLog:   a,
	}
}

func (h *ItemCategoryHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/assets/categories", security.Authorize("user"), h.GetItemCategories)
	router.POST("/assets/categories", security.Authorize("moderator"), h.CreateItemCategory)
	router.DELETE("/assets/categories/:id", security.Authorize("moderator"), h.RemoveItemCategory)
	router.PATCH("/assets/categories/:id", security.Authorize("admin"), h.UpdateItemCategory)

}

func (h *ItemCategoryHandler) GetItemCategories(c *gin.Context) {
	itemCategories, err := h.repository.GetCategories()

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create asset", "details": err.Error()})
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
	req.GenerateNameFromLabel()
	req.GeneratePyrID()
	itemCategory, err := h.repository.PersistItemCategory(req)

	if err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create asset"})
		return
	}
	c.JSON(http.StatusCreated, itemCategory)
}

func (h *ItemCategoryHandler) RemoveItemCategory(c *gin.Context) {
	CategoryID := c.Param("id")

	if _, err := strconv.Atoi(CategoryID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id parameter, must be an integer"})
		return
	}

	hasRelatedItems := h.ar.HasRelatedItems(CategoryID)
	hasRelatedItemsInStock := h.sr.HasRelatedItems(CategoryID)

	if hasRelatedItems || hasRelatedItemsInStock {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "cannot delete category with id " + CategoryID + ": related items exist"})
		return
	}

	err := h.repository.DeleteItemCategoryByID(CategoryID)
	if err != nil {
		if _, ok := err.(*custom_error.ForeignKeyViolationError); ok {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Nie mona usunąć kategorii #" + CategoryID + ": istnieje powiązany sprzęt"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete asset category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item category deleted successfully"})
}

func (h *ItemCategoryHandler) UpdateItemCategory(c *gin.Context) {
	var req models.PatchItemCategoryRequest

	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URI parameters", "details": err.Error()})
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nie poprawne dane żądania", "details": err.Error(), "code": "invalid_request_payload"})
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

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	err := h.repository.UpdateItemCategory(req.ID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update item category", "details": err.Error()})
		return
	}

	go h.AuditLog.Log(
		"update",
		map[string]interface{}{
			"category_id": req.ID,
			"msg":         "Item category updated successfully",
		},
		&models.ItemCategory{ID: req.ID},
	)

	c.JSON(http.StatusOK, gin.H{"message": "Item category updated successfully"})
}
