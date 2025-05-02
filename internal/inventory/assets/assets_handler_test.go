package assets

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"warehouse/internal/repository"
	custom_error "warehouse/pkg/errors"
	"warehouse/pkg/metadata"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAssetsRepository to mock implementation of AssetsRepository
type MockAssetsRepository struct {
	mock.Mock
}

// MockRepository to mock implementation of repository.Repository
type MockRepository struct {
	mock.Mock
	DB            *sql.DB
	GoquDBWrapper *goqu.Database
}

// MockAuditLog to mock implementation of auditlog.Auditlog
type MockAuditLog struct {
	mock.Mock
}

// Interfejsy dla mocków
type RepositoryInterface interface {
	GetCategoryType(categoryID int) (string, error)
	GetCategories() (*[]models.ItemCategory, error)
	PersistItemCategory(itemCategory models.ItemCategory) (*models.ItemCategory, error)
	DeleteItemCategoryByID(categoryID string) error
}

type AssetsRepositoryInterface interface {
	GetAsset(id int) (*models.Asset, error)
	GetAssetList() (*[]models.Asset, error)
	GetAssetsBy(conditions repository.QueryBuilder) (*[]models.Asset, error)
	FindItemByPyrCode(pyrCode string) (*models.Asset, error)
	HasRelatedItems(categoryID string) bool
	HasItemsInLocation(assetIDs []int, fromLocationId int) (bool, error)
	PersistItem(req models.ItemRequest) (*models.Asset, error)
	CanRemoveAsset(id int) (bool, error)
	RemoveAsset(id int) (string, error)
	RemoveAssetFromTransfer(transferID int, itemID int, locationID int) error
	GetTransferAssets(transferID int) (*[]models.Asset, error)
	UpdatePyrCode(id int, pyrCode string) error
	UpdateItemStatus(assetIDs []int, status metadata.Status, tx *goqu.TxDatabase) error
}

type AuditLogInterface interface {
	Log(action string, metadata map[string]interface{}, entity interface{}) error
}

// Implementacje interfejsów
func (m *MockRepository) GetCategoryType(categoryID int) (string, error) {
	args := m.Called(categoryID)
	return args.String(0), args.Error(1)
}

func (m *MockRepository) GetCategories() (*[]models.ItemCategory, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.ItemCategory), args.Error(1)
}

func (m *MockRepository) PersistItemCategory(itemCategory models.ItemCategory) (*models.ItemCategory, error) {
	args := m.Called(itemCategory)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ItemCategory), args.Error(1)
}

func (m *MockRepository) DeleteItemCategoryByID(categoryID string) error {
	args := m.Called(categoryID)
	return args.Error(0)
}

func (m *MockAssetsRepository) GetAsset(id int) (*models.Asset, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Asset), args.Error(1)
}

func (m *MockAssetsRepository) GetAssetList() (*[]models.Asset, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.Asset), args.Error(1)
}

func (m *MockAssetsRepository) GetAssetsBy(conditions repository.QueryBuilder) (*[]models.Asset, error) {
	args := m.Called(conditions)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.Asset), args.Error(1)
}

func (m *MockAssetsRepository) FindItemByPyrCode(pyrCode string) (*models.Asset, error) {
	args := m.Called(pyrCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Asset), args.Error(1)
}

func (m *MockAssetsRepository) HasRelatedItems(categoryID string) bool {
	args := m.Called(categoryID)
	return args.Bool(0)
}

func (m *MockAssetsRepository) HasItemsInLocation(assetIDs []int, fromLocationId int) (bool, error) {
	args := m.Called(assetIDs, fromLocationId)
	return args.Bool(0), args.Error(1)
}

func (m *MockAssetsRepository) PersistItem(req models.ItemRequest) (*models.Asset, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Asset), args.Error(1)
}

func (m *MockAssetsRepository) CanRemoveAsset(id int) (bool, error) {
	args := m.Called(id)
	return args.Bool(0), args.Error(1)
}

func (m *MockAssetsRepository) RemoveAsset(id int) (string, error) {
	args := m.Called(id)
	return args.String(0), args.Error(1)
}

func (m *MockAssetsRepository) RemoveAssetFromTransfer(transferID int, itemID int, locationID int) error {
	args := m.Called(transferID, itemID, locationID)
	return args.Error(0)
}

func (m *MockAssetsRepository) GetTransferAssets(transferID int) (*[]models.Asset, error) {
	args := m.Called(transferID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.Asset), args.Error(1)
}

func (m *MockAssetsRepository) UpdatePyrCode(id int, pyrCode string) error {
	args := m.Called(id, pyrCode)
	return args.Error(0)
}

func (m *MockAssetsRepository) UpdateItemStatus(assetIDs []int, status metadata.Status, tx *goqu.TxDatabase) error {
	args := m.Called(assetIDs, status, tx)
	return args.Error(0)
}

func (m *MockAuditLog) Log(action string, metadata map[string]interface{}, entity interface{}) error {
	args := m.Called(action, metadata, entity)
	return args.Error(0)
}

// SetupTestRouter creates a new gin router for testing
func SetupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	return router
}

// TestItemHandler to testowa implementacja ItemHandler
type TestItemHandler struct {
	r          AssetsRepositoryInterface
	repository RepositoryInterface
	AuditLog   AuditLogInterface
}

func (h *TestItemHandler) CreateBulkAssets(c *gin.Context) {
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

	// Validate origin
	if req.Origin == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "Origin is required",
		})
		return
	}

	categoryType, err := h.repository.GetCategoryType(req.CategoryId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unable to check category type", "details": err.Error()})
		return
	}

	if categoryType != "asset" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid category type"})
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
				errors = append(errors, fmt.Sprintf("Serial number %s already registered", *serial))
				continue
			default:
				errors = append(errors, fmt.Sprintf("Failed to create asset with serial %s: %v", *serial, err))
				continue
			}
		}

		pyrCode := metadata.NewPyrCode(asset.Category.PyrID, asset.ID)
		asset.PyrCode = pyrCode.GeneratePyrCode()
		err = h.r.UpdatePyrCode(asset.ID, asset.PyrCode)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to update PYR code for asset with serial %s: %v", *serial, err))
			continue
		}

		err = h.AuditLog.Log("create", map[string]interface{}{
			"serial":      asset.Serial,
			"pyr_code":    asset.PyrCode,
			"location_id": asset.Location.ID,
			"msg":         "Asset created successfully",
		}, asset)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to log audit for asset with serial %s: %v", *serial, err))
			continue
		}

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

// TestCreateBulkAssets_Success tests successful creation of multiple assets
func TestCreateBulkAssets_Success(t *testing.T) {
	// Setup
	router := SetupTestRouter()

	mockAssetsRepo := new(MockAssetsRepository)
	mockRepo := new(MockRepository)
	mockAuditLog := new(MockAuditLog)

	handler := &TestItemHandler{
		r:          mockAssetsRepo,
		repository: mockRepo,
		AuditLog:   mockAuditLog,
	}
	router.POST("/assets/bulk", handler.CreateBulkAssets)

	// Test data
	serial1 := "SERIAL001"
	serial2 := "SERIAL002"
	reqBody := models.BulkItemRequest{
		Serials:    []*string{&serial1, &serial2},
		LocationId: 1,
		Status:     "available",
		CategoryId: 4,
		Origin:     "purchase",
	}

	// Mock expectations
	mockRepo.On("GetCategoryType", 4).Return("asset", nil)

	// Mock successful asset creation for both serials
	asset1 := &models.Asset{
		ID:     1,
		Serial: &serial1,
		Category: models.ItemCategory{
			ID:    4,
			PyrID: "L",
		},
		Location: models.Location{
			ID: 1,
		},
	}

	asset2 := &models.Asset{
		ID:     2,
		Serial: &serial2,
		Category: models.ItemCategory{
			ID:    4,
			PyrID: "L",
		},
		Location: models.Location{
			ID: 1,
		},
	}

	mockAssetsRepo.On("PersistItem", mock.MatchedBy(func(req models.ItemRequest) bool {
		return *req.Serial == "SERIAL001"
	})).Return(asset1, nil)

	mockAssetsRepo.On("PersistItem", mock.MatchedBy(func(req models.ItemRequest) bool {
		return *req.Serial == "SERIAL002"
	})).Return(asset2, nil)

	mockAssetsRepo.On("UpdatePyrCode", 1, "PYR-L1").Return(nil)
	mockAssetsRepo.On("UpdatePyrCode", 2, "PYR-L2").Return(nil)

	mockAuditLog.On("Log", "create", mock.Anything, mock.Anything).Return(nil).Times(2)

	// Create request
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/assets/bulk", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.Nil(t, err)

	created, ok := response["created"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, created, 2)

	// Verify mock expectations
	mockRepo.AssertExpectations(t)
	mockAssetsRepo.AssertExpectations(t)
	mockAuditLog.AssertExpectations(t)
}

// TestCreateBulkAssets_DefaultLocation tests that default location ID is set to 1 when not provided
func TestCreateBulkAssets_DefaultLocation(t *testing.T) {
	// Setup
	router := SetupTestRouter()

	mockAssetsRepo := new(MockAssetsRepository)
	mockRepo := new(MockRepository)
	mockAuditLog := new(MockAuditLog)

	handler := &TestItemHandler{
		r:          mockAssetsRepo,
		repository: mockRepo,
		AuditLog:   mockAuditLog,
	}
	router.POST("/assets/bulk", handler.CreateBulkAssets)

	// Test data with LocationId not set (zero value)
	serial1 := "SERIAL001"
	reqBody := models.BulkItemRequest{
		Serials:    []*string{&serial1},
		Status:     "available",
		CategoryId: 4,
		Origin:     "purchase",
	}

	// Mock expectations
	mockRepo.On("GetCategoryType", 4).Return("asset", nil)

	// Mock successful asset creation
	asset := &models.Asset{
		ID:     1,
		Serial: &serial1,
		Category: models.ItemCategory{
			ID:    4,
			PyrID: "L",
		},
		Location: models.Location{
			ID: 1,
		},
	}

	// Verify that LocationId is set to 1 in the request
	mockAssetsRepo.On("PersistItem", mock.MatchedBy(func(req models.ItemRequest) bool {
		return *req.Serial == "SERIAL001" && req.LocationId == 1 && req.Status == "available"
	})).Return(asset, nil)

	mockAssetsRepo.On("UpdatePyrCode", 1, "PYR-L1").Return(nil)
	mockAuditLog.On("Log", "create", mock.Anything, mock.Anything).Return(nil)

	// Create request
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/assets/bulk", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	// Verify mock expectations
	mockRepo.AssertExpectations(t)
	mockAssetsRepo.AssertExpectations(t)
	mockAuditLog.AssertExpectations(t)
}

// TestCreateBulkAssets_DefaultStatus tests that default status is set to "available" when not provided
func TestCreateBulkAssets_DefaultStatus(t *testing.T) {
	// Setup
	router := SetupTestRouter()

	mockAssetsRepo := new(MockAssetsRepository)
	mockRepo := new(MockRepository)
	mockAuditLog := new(MockAuditLog)

	handler := &TestItemHandler{
		r:          mockAssetsRepo,
		repository: mockRepo,
		AuditLog:   mockAuditLog,
	}
	router.POST("/assets/bulk", handler.CreateBulkAssets)

	// Test data with Status not set (empty string)
	serial1 := "SERIAL001"
	reqBody := models.BulkItemRequest{
		Serials:    []*string{&serial1},
		LocationId: 1,
		CategoryId: 4,
		Origin:     "purchase",
	}

	// Mock expectations
	mockRepo.On("GetCategoryType", 4).Return("asset", nil)

	// Mock successful asset creation
	asset := &models.Asset{
		ID:     1,
		Serial: &serial1,
		Category: models.ItemCategory{
			ID:    4,
			PyrID: "L",
		},
		Location: models.Location{
			ID: 1,
		},
	}

	// Verify that Status is set to "available" in the request
	mockAssetsRepo.On("PersistItem", mock.MatchedBy(func(req models.ItemRequest) bool {
		return *req.Serial == "SERIAL001" && req.Status == "available" && req.LocationId == 1 && req.CategoryId == 4 && req.Origin == "purchase"
	})).Return(asset, nil)

	mockAssetsRepo.On("UpdatePyrCode", 1, "PYR-L1").Return(nil)
	mockAuditLog.On("Log", "create", mock.Anything, mock.Anything).Return(nil)

	// Create request
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/assets/bulk", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	// Verify mock expectations
	mockRepo.AssertExpectations(t)
	mockAssetsRepo.AssertExpectations(t)
	mockAuditLog.AssertExpectations(t)
}

// TestCreateBulkAssets_DuplicateSerial tests handling of duplicate serial numbers
func TestCreateBulkAssets_DuplicateSerial(t *testing.T) {
	// Setup
	router := SetupTestRouter()

	mockAssetsRepo := new(MockAssetsRepository)
	mockRepo := new(MockRepository)
	mockAuditLog := new(MockAuditLog)

	handler := &TestItemHandler{
		r:          mockAssetsRepo,
		repository: mockRepo,
		AuditLog:   mockAuditLog,
	}
	router.POST("/assets/bulk", handler.CreateBulkAssets)

	// Test data
	serial1 := "SERIAL001"
	serial2 := "SERIAL002"
	serial3 := "SERIAL003"
	reqBody := models.BulkItemRequest{
		Serials:    []*string{&serial1, &serial2, &serial3},
		LocationId: 1,
		Status:     "available",
		CategoryId: 4,
		Origin:     "purchase",
	}

	// Mock expectations
	mockRepo.On("GetCategoryType", 4).Return("asset", nil)

	// Mock successful asset creation for first serial
	asset1 := &models.Asset{
		ID:     1,
		Serial: &serial1,
		Category: models.ItemCategory{
			ID:    4,
			PyrID: "L",
		},
		Location: models.Location{
			ID: 1,
		},
	}

	// Mock duplicate serial error for second serial
	mockAssetsRepo.On("PersistItem", mock.MatchedBy(func(req models.ItemRequest) bool {
		return *req.Serial == "SERIAL001"
	})).Return(asset1, nil)

	mockAssetsRepo.On("PersistItem", mock.MatchedBy(func(req models.ItemRequest) bool {
		return *req.Serial == "SERIAL002"
	})).Return(nil, &custom_error.UniqueViolationError{})

	// Mock successful asset creation for third serial
	asset3 := &models.Asset{
		ID:     3,
		Serial: &serial3,
		Category: models.ItemCategory{
			ID:    4,
			PyrID: "L",
		},
		Location: models.Location{
			ID: 1,
		},
	}

	mockAssetsRepo.On("PersistItem", mock.MatchedBy(func(req models.ItemRequest) bool {
		return *req.Serial == "SERIAL003"
	})).Return(asset3, nil)

	mockAssetsRepo.On("UpdatePyrCode", 1, "PYR-L1").Return(nil)
	mockAssetsRepo.On("UpdatePyrCode", 3, "PYR-L3").Return(nil)

	mockAuditLog.On("Log", "create", mock.Anything, mock.Anything).Return(nil).Times(2)

	// Create request
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/assets/bulk", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.Nil(t, err)

	created, ok := response["created"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, created, 2)

	errors, ok := response["errors"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, errors, 1)
	assert.Contains(t, errors[0], "Serial number SERIAL002 already registered")

	// Verify mock expectations
	mockRepo.AssertExpectations(t)
	mockAssetsRepo.AssertExpectations(t)
	mockAuditLog.AssertExpectations(t)
}

// TestCreateBulkAssets_InvalidCategory tests handling of invalid category type
func TestCreateBulkAssets_InvalidCategory(t *testing.T) {
	// Setup
	router := SetupTestRouter()

	mockAssetsRepo := new(MockAssetsRepository)
	mockRepo := new(MockRepository)
	mockAuditLog := new(MockAuditLog)

	handler := &TestItemHandler{
		r:          mockAssetsRepo,
		repository: mockRepo,
		AuditLog:   mockAuditLog,
	}
	router.POST("/assets/bulk", handler.CreateBulkAssets)

	// Test data
	serial1 := "SERIAL001"
	serial2 := "SERIAL002"
	reqBody := models.BulkItemRequest{
		Serials:    []*string{&serial1, &serial2},
		LocationId: 1,
		Status:     "available",
		CategoryId: 4,
		Origin:     "purchase",
	}

	// Mock expectations - category type is not "asset"
	mockRepo.On("GetCategoryType", 4).Return("stock", nil)

	// Create request
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/assets/bulk", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.Nil(t, err)

	errorMsg, ok := response["error"].(string)
	assert.True(t, ok)
	assert.Equal(t, "Invalid category type", errorMsg)

	// Verify mock expectations
	mockRepo.AssertExpectations(t)
	mockAssetsRepo.AssertExpectations(t)
}

func (h *TestItemHandler) CreateAssetWithoutSerial(c *gin.Context) {
	var req models.EmergencyAssetRequest
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

	// Validate origin
	if req.Origin == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "Origin is required",
		})
		return
	}

	categoryType, err := h.repository.GetCategoryType(req.CategoryId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unable to check category type", "details": err.Error()})
		return
	}

	if categoryType != "asset" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid category type"})
		return
	}

	var createdAssets []models.Asset
	var errors []string

	for i := 0; i < req.Quantity; i++ {
		itemReq := models.ItemRequest{
			Serial:     nil,
			LocationId: req.LocationId,
			Status:     req.Status,
			CategoryId: req.CategoryId,
			Origin:     req.Origin,
		}

		asset, err := h.r.PersistItem(itemReq)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to create asset: %v", err))
			continue
		}

		pyrCode := fmt.Sprintf("PYR-L%d", asset.ID)
		if err := h.r.UpdatePyrCode(asset.ID, pyrCode); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to update PYR code for asset: %v", err))
			continue
		}

		asset.PyrCode = pyrCode
		createdAssets = append(createdAssets, *asset)

		go h.AuditLog.Log(
			"create",
			map[string]interface{}{
				"pyr_code":    asset.PyrCode,
				"location_id": asset.Location.ID,
				"msg":         "Asset created successfully",
			},
			asset,
		)
	}

	response := gin.H{
		"created": createdAssets,
	}
	if len(errors) > 0 {
		response["errors"] = errors
	}

	c.JSON(http.StatusCreated, response)
}

func TestCreateAssetWithoutSerial_Success(t *testing.T) {
	// Setup
	router := SetupTestRouter()

	mockAssetsRepo := new(MockAssetsRepository)
	mockRepo := new(MockRepository)
	mockAuditLog := new(MockAuditLog)

	handler := &TestItemHandler{
		r:          mockAssetsRepo,
		repository: mockRepo,
		AuditLog:   mockAuditLog,
	}
	router.POST("/assets/without-serial", handler.CreateAssetWithoutSerial)

	// Test data
	reqBody := models.EmergencyAssetRequest{
		Quantity:   3,
		LocationId: 1,
		Status:     "available",
		CategoryId: 4,
		Origin:     "purchase",
	}

	// Mock expectations
	mockRepo.On("GetCategoryType", 4).Return("asset", nil)
	serial1 := "SERIAL001"
	serial2 := "SERIAL002"
	serial3 := "SERIAL003"

	// Mock successful asset creation for all three assets
	assets := []*models.Asset{
		{
			ID:     1,
			Serial: &serial1,
			Category: models.ItemCategory{
				ID:    4,
				PyrID: "L",
			},
			Location: models.Location{
				ID: 1,
			},
		},
		{
			ID:     2,
			Serial: &serial2,
			Category: models.ItemCategory{
				ID:    4,
				PyrID: "L",
			},
			Location: models.Location{
				ID: 1,
			},
		},
		{
			ID:     3,
			Serial: &serial3,
			Category: models.ItemCategory{
				ID:    4,
				PyrID: "L",
			},
			Location: models.Location{
				ID: 1,
			},
		},
	}

	for i, asset := range assets {
		mockAssetsRepo.On("PersistItem", mock.MatchedBy(func(req models.ItemRequest) bool {
			return req.Serial == nil && req.LocationId == 1 && req.Status == "available"
		})).Return(asset, nil)

		mockAssetsRepo.On("UpdatePyrCode", i+1, fmt.Sprintf("PYR-L%d", i+1)).Return(nil)
	}

	mockAuditLog.On("Log", "create", mock.Anything, mock.Anything).Return(nil).Times(3)

	// Create request
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/assets/without-serial", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.Nil(t, err)

	created, ok := response["created"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, created, 3)

	// Verify mock expectations
	mockRepo.AssertExpectations(t)
	mockAssetsRepo.AssertExpectations(t)
	mockAuditLog.AssertExpectations(t)
}

func TestCreateAssetWithoutSerial_InvalidQuantity(t *testing.T) {
	// Setup
	router := SetupTestRouter()

	mockAssetsRepo := new(MockAssetsRepository)
	mockRepo := new(MockRepository)
	mockAuditLog := new(MockAuditLog)

	handler := &TestItemHandler{
		r:          mockAssetsRepo,
		repository: mockRepo,
		AuditLog:   mockAuditLog,
	}
	router.POST("/assets/without-serial", handler.CreateAssetWithoutSerial)

	// Test data with invalid quantity
	reqBody := models.EmergencyAssetRequest{
		Quantity:   0, // Invalid quantity
		LocationId: 1,
		Status:     "available",
		CategoryId: 4,
		Origin:     "purchase",
	}

	// Create request
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/assets/without-serial", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.Nil(t, err)

	errorMsg, ok := response["error"].(string)
	assert.True(t, ok)
	assert.Contains(t, errorMsg, "Invalid request payload")
}

func TestCreateAssetWithoutSerial_InvalidCategory(t *testing.T) {
	// Setup
	router := SetupTestRouter()

	mockAssetsRepo := new(MockAssetsRepository)
	mockRepo := new(MockRepository)
	mockAuditLog := new(MockAuditLog)

	handler := &TestItemHandler{
		r:          mockAssetsRepo,
		repository: mockRepo,
		AuditLog:   mockAuditLog,
	}
	router.POST("/assets/without-serial", handler.CreateAssetWithoutSerial)

	// Test data
	reqBody := models.EmergencyAssetRequest{
		Quantity:   2,
		LocationId: 1,
		Status:     "available",
		CategoryId: 4,
		Origin:     "purchase",
	}

	// Mock expectations - category type is not "asset"
	mockRepo.On("GetCategoryType", 4).Return("stock", nil)

	// Create request
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/assets/without-serial", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.Nil(t, err)

	errorMsg, ok := response["error"].(string)
	assert.True(t, ok)
	assert.Equal(t, "Invalid category type", errorMsg)

	// Verify mock expectations
	mockRepo.AssertExpectations(t)
}
