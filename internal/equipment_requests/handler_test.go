package equipment_requests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"warehouse/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockQuestRepository is a mock implementation for testing
type mockQuestRepository struct {
	quests           []Quest
	syncLogs         []SyncLog
	mappings         map[string]int
	categoryMappings []CategoryMapping
	statusUpdate     func(ctx context.Context, questID string, status string) error
	stockFinder      func(locationID, categoryID int) ([]StockMatch, error)
}

// mockTransferCreator is a mock TransferCreator for testing
type mockTransferCreator struct {
	transferID int
	err        error
}

func (m *mockQuestRepository) CreateQuest(ctx context.Context, quest *Quest) error {
	m.quests = append(m.quests, *quest)
	return nil
}

func (m *mockQuestRepository) UpdateQuest(ctx context.Context, questID string, quest *Quest) error {
	for i, q := range m.quests {
		if q.ID == questID {
			m.quests[i] = *quest
			return nil
		}
	}
	return nil
}

func (m *mockQuestRepository) GetQuestByID(ctx context.Context, questID string) (*Quest, error) {
	for _, q := range m.quests {
		if q.ID == questID {
			return &q, nil
		}
	}
	return nil, assert.AnError
}

func (m *mockQuestRepository) GetQuestByKey(ctx context.Context, questKey string) (*Quest, error) {
	for _, q := range m.quests {
		if q.QuestKey == questKey {
			return &q, nil
		}
	}
	return nil, nil
}

func (m *mockQuestRepository) ListQuests(ctx context.Context, filter QuestFilter) ([]Quest, error) {
	result := []Quest{}
	for _, q := range m.quests {
		if filter.Status != "" && q.Status != filter.Status {
			continue
		}
		if filter.LocationID != nil && (q.LocationID == nil || *q.LocationID != *filter.LocationID) {
			continue
		}
		result = append(result, q)
	}

	// Apply pagination
	start := filter.Offset
	end := start + filter.Limit
	if start > len(result) {
		return []Quest{}, nil
	}
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

func (m *mockQuestRepository) UpdateQuestStatus(ctx context.Context, questID string, status string) error {
	if m.statusUpdate != nil {
		return m.statusUpdate(ctx, questID, status)
	}
	for i, q := range m.quests {
		if q.ID == questID {
			m.quests[i].Status = status
			return nil
		}
	}
	return assert.AnError
}

func (m *mockQuestRepository) CreateSyncLog(ctx context.Context, log *SyncLog) error {
	log.ID = len(m.syncLogs) + 1
	m.syncLogs = append(m.syncLogs, *log)
	return nil
}

func (m *mockQuestRepository) GetLatestSyncLog(ctx context.Context) (*SyncLog, error) {
	if len(m.syncLogs) == 0 {
		return nil, assert.AnError
	}
	return &m.syncLogs[len(m.syncLogs)-1], nil
}

func (m *mockQuestRepository) GetCategoryMapping(ctx context.Context, itemName string) (*int, error) {
	if catID, ok := m.mappings[itemName]; ok {
		return &catID, nil
	}
	return nil, nil
}

func (m *mockQuestRepository) CreateCategoryMapping(ctx context.Context, mapping *CategoryMapping) error {
	mapping.ID = len(m.mappings) + 1
	mapping.CreatedAt = time.Now()
	m.mappings[mapping.FormItemName] = mapping.CategoryID
	return nil
}

func (m *mockQuestRepository) IncrementMappingUsage(ctx context.Context, itemName string) error {
	return nil
}

func (m *mockQuestRepository) AddTransferToQuest(ctx context.Context, questID string, transferID int) error {
	for i, q := range m.quests {
		if q.ID == questID {
			m.quests[i].Transfers = append(m.quests[i].Transfers, QuestTransfer{TransferID: transferID, Status: "in_transit"})
			m.quests[i].Status = "in_progress"
			return nil
		}
	}
	return assert.AnError
}

func (m *mockQuestRepository) GetQuestByTransferID(ctx context.Context, transferID int) (*Quest, error) {
	for _, q := range m.quests {
		for _, t := range q.Transfers {
			if t.TransferID == transferID {
				return &q, nil
			}
		}
	}
	return nil, nil
}

func (m *mockQuestRepository) RemoveTransferFromQuest(ctx context.Context, transferID int) error {
	for i, q := range m.quests {
		for j, t := range q.Transfers {
			if t.TransferID == transferID {
				m.quests[i].Transfers = append(m.quests[i].Transfers[:j], m.quests[i].Transfers[j+1:]...)
				if len(m.quests[i].Transfers) == 0 {
					m.quests[i].Status = "pending"
				}
				return nil
			}
		}
	}
	return nil
}

func (m *mockQuestRepository) GetActiveTransfersForQuest(ctx context.Context, questID string) ([]int, error) {
	for _, q := range m.quests {
		if q.ID == questID {
			var ids []int
			for _, t := range q.Transfers {
				if t.Status != "completed" && t.Status != "cancelled" {
					ids = append(ids, t.TransferID)
				}
			}
			return ids, nil
		}
	}
	return nil, nil
}

func (m *mockQuestRepository) FindStockItemsByCategory(fromLocationID int, categoryID int) ([]StockMatch, error) {
	if m.stockFinder != nil {
		return m.stockFinder(fromLocationID, categoryID)
	}
	return nil, nil
}

func (m *mockQuestRepository) ResolveLocationByPavilionAndName(pavilion, name string) (*int, error) {
	return nil, nil
}

func (m *mockQuestRepository) ResolveLocationByNameOnly(name string) (*int, error) {
	return nil, nil
}

func (m *mockQuestRepository) GetLocationMapping(ctx context.Context, pavilion, locationName string) (*int, error) {
	return nil, nil
}

func (m *mockQuestRepository) CreateLocationMapping(ctx context.Context, mapping *LocationMapping) error {
	return nil
}

func (m *mockQuestRepository) ListLocationMappings(ctx context.Context) ([]LocationMapping, error) {
	return nil, nil
}

func (m *mockQuestRepository) DeleteLocationMapping(ctx context.Context, id int) error {
	return nil
}

func (m *mockQuestRepository) IncrementLocationMappingUsage(ctx context.Context, pavilion, locationName string) error {
	return nil
}

func (m *mockQuestRepository) UpdateQuestLocationResolution(ctx context.Context, questID string, locationID *int, resolved bool) error {
	for i, q := range m.quests {
		if q.ID == questID {
			m.quests[i].LocationID = locationID
			m.quests[i].LocationResolved = resolved
			return nil
		}
	}
	return assert.AnError
}

func (m *mockQuestRepository) ListUnresolvedLocationQuests(ctx context.Context) ([]Quest, error) {
	result := []Quest{}
	for _, q := range m.quests {
		if !q.LocationResolved {
			result = append(result, q)
		}
	}
	return result, nil
}

func (m *mockQuestRepository) ListCategoryMappings(ctx context.Context) ([]CategoryMapping, error) {
	return m.categoryMappings, nil
}

func (m *mockQuestRepository) DeleteCategoryMapping(ctx context.Context, id int) error {
	for i, cm := range m.categoryMappings {
		if cm.ID == id {
			m.categoryMappings = append(m.categoryMappings[:i], m.categoryMappings[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("category mapping %d not found", id)
}

func (m *mockTransferCreator) InitTransfer(req models.TransferRequest, status string) (int, error) {
	return m.transferID, m.err
}

func setupTestHandler() (*Handler, *mockQuestRepository) {
	mockRepo := &mockQuestRepository{
		quests:   []Quest{},
		syncLogs: []SyncLog{},
		mappings: make(map[string]int),
	}

	// Service doesn't need questRepo for these handler tests
	// Handler uses service.questRepo directly
	service := &Service{}
	handler := &Handler{service: service}

	// Inject mock repo into service
	service.questRepo = mockRepo

	return handler, mockRepo
}

func TestHandler_GetQuest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mockRepo := setupTestHandler()

	// Add test quest to mock repository
	testQuest := Quest{
		ID:       "quest-test123",
		QuestKey: "test-key",
		Destination: Destination{
			Pavilion: "PCC",
			Location: "Maskarada",
		},
		Recipient:    "Jan Kowalski",
		DeliveryDate: "2025-06-13",
		Items: []QuestItem{
			{Name: "Laptop", Quantity: 2, CategoryMatch: "exact"},
		},
		Status:     "pending",
		SourceRows: []int{10},
		LastSynced: time.Now(),
	}
	mockRepo.quests = append(mockRepo.quests, testQuest)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "quest-test123"}}
	c.Request = httptest.NewRequest("GET", "/api/equipment-requests/quests/quest-test123", nil)

	// Execute handler
	handler.GetQuest(c)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response Quest
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, testQuest.ID, response.ID)
	assert.Equal(t, testQuest.Recipient, response.Recipient)
}

func TestHandler_ListQuests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mockRepo := setupTestHandler()

	// Add test quests
	mockRepo.quests = []Quest{
		{
			ID:           "quest-1",
			QuestKey:     "key-1",
			Destination:  Destination{Pavilion: "P1", Location: "L1"},
			Recipient:    "User 1",
			DeliveryDate: "2025-06-13",
			Items:        []QuestItem{{Name: "Item 1", Quantity: 1, CategoryMatch: "none"}},
			Status:       "pending",
			SourceRows:   []int{1},
			LastSynced:   time.Now(),
		},
		{
			ID:           "quest-2",
			QuestKey:     "key-2",
			Destination:  Destination{Pavilion: "P2", Location: "L2"},
			Recipient:    "User 2",
			DeliveryDate: "2025-06-14",
			Items:        []QuestItem{{Name: "Item 2", Quantity: 1, CategoryMatch: "none"}},
			Status:       "in_progress",
			SourceRows:   []int{2},
			LastSynced:   time.Now(),
		},
	}

	tests := []struct {
		name           string
		queryParams    string
		expectedCount  int
		expectedStatus int
	}{
		{
			name:           "List all quests",
			queryParams:    "",
			expectedCount:  2,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Filter by status",
			queryParams:    "?status=pending",
			expectedCount:  1,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Pagination with limit",
			queryParams:    "?limit=1",
			expectedCount:  1,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Pagination with offset",
			queryParams:    "?limit=10&offset=1",
			expectedCount:  1,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/equipment-requests/quests"+tt.queryParams, nil)

			handler.ListQuests(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			quests := response["quests"].([]interface{})
			assert.Equal(t, tt.expectedCount, len(quests))
		})
	}
}

func TestHandler_UpdateQuestStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		questID        string
		requestBody    map[string]string
		expectedStatus int
		expectedError  bool
	}{
		{
			name:           "Valid status update",
			questID:        "quest-test123",
			requestBody:    map[string]string{"status": "in_progress"},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:           "Invalid status",
			questID:        "quest-test123",
			requestBody:    map[string]string{"status": "invalid_status"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:           "Missing status field",
			questID:        "quest-test123",
			requestBody:    map[string]string{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockRepo := setupTestHandler()

			// Add test quest
			mockRepo.quests = append(mockRepo.quests, Quest{
				ID:         tt.questID,
				Status:     "pending",
				SourceRows: []int{1},
			})

			// Create request
			bodyBytes, _ := json.Marshal(tt.requestBody)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "id", Value: tt.questID}}
			c.Request = httptest.NewRequest("PATCH", "/api/equipment-requests/quests/"+tt.questID+"/status", bytes.NewReader(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.UpdateQuestStatus(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectedError {
				assert.Contains(t, response, "error")
			} else {
				assert.Contains(t, response, "message")
			}
		})
	}
}

func TestHandler_CreateCategoryMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := setupTestHandler()

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Valid mapping",
			requestBody: map[string]interface{}{
				"form_item_name": "Laptop Dell",
				"category_id":    float64(100),
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "With created_by",
			requestBody: map[string]interface{}{
				"form_item_name": "Mouse Logitech",
				"category_id":    float64(200),
				"created_by":     float64(7),
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Missing required field",
			requestBody: map[string]interface{}{
				"form_item_name": "Keyboard",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.requestBody)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/api/equipment-requests/category-mapping", bytes.NewReader(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.CreateCategoryMapping(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_GetSyncLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mockRepo := setupTestHandler()

	// Add sync log
	syncLog := SyncLog{
		ID:              1,
		SyncedAt:        time.Now(),
		RowsProcessed:   20,
		QuestsCreated:   5,
		QuestsUpdated:   3,
		QuestsUnchanged: 12,
		ItemsAdded:      8,
		ItemsRemoved:    2,
		Success:         true,
		DurationMs:      2500,
		SheetID:         "test-sheet-id",
	}
	mockRepo.syncLogs = append(mockRepo.syncLogs, syncLog)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/equipment-requests/sync-log", nil)

	handler.GetSyncLog(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SyncLog
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, syncLog.QuestsCreated, response.QuestsCreated)
	assert.Equal(t, syncLog.QuestsUpdated, response.QuestsUpdated)
}

func TestGetIntQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		queryValue    string
		defaultValue  int
		expectedValue int
	}{
		{
			name:          "Valid integer",
			queryValue:    "42",
			defaultValue:  10,
			expectedValue: 42,
		},
		{
			name:          "Empty string - use default",
			queryValue:    "",
			defaultValue:  10,
			expectedValue: 10,
		},
		{
			name:          "Invalid integer - use default",
			queryValue:    "invalid",
			defaultValue:  10,
			expectedValue: 10,
		},
		{
			name:          "Negative integer",
			queryValue:    "-5",
			defaultValue:  10,
			expectedValue: -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/?test="+tt.queryValue, nil)

			result := getIntQuery(c, "test", tt.defaultValue)
			assert.Equal(t, tt.expectedValue, result)
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "Item exists",
			slice:    []string{"pending", "in_progress", "completed"},
			item:     "in_progress",
			expected: true,
		},
		{
			name:     "Item does not exist",
			slice:    []string{"pending", "in_progress", "completed"},
			item:     "cancelled",
			expected: false,
		},
		{
			name:     "Empty slice",
			slice:    []string{},
			item:     "pending",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// Phase 4 Tests
// ============================================================================

func TestHandler_UpdateQuestStatus_409_WhenQuestHasTransfer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mockRepo := setupTestHandler()

	mockRepo.quests = append(mockRepo.quests, Quest{
		ID:         "quest-linked",
		Status:     "in_progress",
		Transfers:  []QuestTransfer{{TransferID: 42, Status: "in_transit"}},
		SourceRows: []int{1},
	})

	bodyBytes, _ := json.Marshal(map[string]string{"status": "completed"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "quest-linked"}}
	c.Request = httptest.NewRequest("PATCH", "/api/equipment-requests/quests/quest-linked/status", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateQuestStatus(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp, "error")
	assert.Contains(t, resp["error"], "active transfers")
}

func setupHandlerWithTransferCreator(tc TransferCreator) (*Handler, *mockQuestRepository) {
	mockRepo := &mockQuestRepository{
		quests:   []Quest{},
		syncLogs: []SyncLog{},
		mappings: make(map[string]int),
	}
	svc := &Service{transferCreator: tc}
	svc.questRepo = mockRepo
	svc.sseClients = make(map[chan QuestEvent]struct{})
	return &Handler{service: svc}, mockRepo
}

func TestHandler_CreateTransferFromQuest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	toLocationID := 5

	tests := []struct {
		name           string
		questID        string
		quest          *Quest
		body           interface{}
		transferID     int
		transferErr    error
		expectedStatus int
	}{
		{
			name:           "Quest not found returns 404",
			questID:        "quest-missing",
			quest:          nil,
			body:           map[string]interface{}{"from_location_id": 1},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:    "Quest completed returns 409",
			questID: "quest-done",
			quest: &Quest{
				ID:         "quest-done",
				Status:     "completed",
				SourceRows: []int{1},
			},
			body:           map[string]interface{}{"from_location_id": 1, "to_location_id": toLocationID, "stock_items": []map[string]interface{}{{"id": 10, "quantity": 2}}},
			expectedStatus: http.StatusConflict,
		},
		{
			name:    "Quest in_progress allows second transfer",
			questID: "quest-in-progress",
			quest: &Quest{
				ID:        "quest-in-progress",
				Status:    "in_progress",
				Transfers: []QuestTransfer{{TransferID: 99, Status: "in_transit"}},
				SourceRows: []int{1},
			},
			body:           map[string]interface{}{"from_location_id": 1, "to_location_id": toLocationID, "stock_items": []map[string]interface{}{{"id": 10, "quantity": 2}}},
			transferID:     200,
			expectedStatus: http.StatusCreated,
		},
		{
			name:    "Missing from_location_id returns 400",
			questID: "quest-ok",
			quest: &Quest{
				ID:         "quest-ok",
				Status:     "pending",
				SourceRows: []int{1},
			},
			body:           map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "Success — explicit stock items and to_location_id",
			questID: "quest-pending",
			quest: &Quest{
				ID:         "quest-pending",
				Status:     "pending",
				SourceRows: []int{1},
			},
			body:           map[string]interface{}{"from_location_id": 1, "to_location_id": toLocationID, "stock_items": []map[string]interface{}{{"id": 10, "quantity": 2}}},
			transferID:     156,
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &mockTransferCreator{transferID: tt.transferID, err: tt.transferErr}
			handler, mockRepo := setupHandlerWithTransferCreator(tc)

			if tt.quest != nil {
				mockRepo.quests = append(mockRepo.quests, *tt.quest)
			}

			bodyBytes, _ := json.Marshal(tt.body)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "id", Value: tt.questID}}
			c.Request = httptest.NewRequest("POST", "/api/equipment-requests/quests/"+tt.questID+"/transfer", bytes.NewReader(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.CreateTransferFromQuest(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				assert.EqualValues(t, tt.transferID, resp["transfer_id"])
				assert.Equal(t, tt.questID, resp["quest_id"])
			}
		})
	}
}

func TestHandler_PreviewTransferFromQuest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		questID        string
		fromLocationID string
		quest          *Quest
		expectedStatus int
	}{
		{
			name:           "Missing from_location_id returns 400",
			questID:        "quest-1",
			fromLocationID: "",
			quest:          &Quest{ID: "quest-1", Status: "pending", SourceRows: []int{1}},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Quest not found returns 404",
			questID:        "quest-missing",
			fromLocationID: "1",
			quest:          nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Success returns 200 with preview",
			questID:        "quest-2",
			fromLocationID: "1",
			quest: &Quest{
				ID:          "quest-2",
				Status:      "pending",
				Destination: Destination{Pavilion: "P1", Location: "L1"},
				Items:       []QuestItem{{Name: "Laptop", Quantity: 2}},
				SourceRows:  []int{1},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockRepo := setupTestHandler()

			if tt.quest != nil {
				mockRepo.quests = append(mockRepo.quests, *tt.quest)
			}

			url := "/api/equipment-requests/quests/" + tt.questID + "/transfer-preview"
			if tt.fromLocationID != "" {
				url += "?from_location_id=" + tt.fromLocationID
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "id", Value: tt.questID}}
			c.Request = httptest.NewRequest("GET", url, nil)

			handler.PreviewTransferFromQuest(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var resp TransferPreview
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				assert.Equal(t, 1, resp.FromLocationID)
			}
		})
	}
}

func TestService_OnTransferStatusChanged(t *testing.T) {
	transferID := 10
	questID := "quest-linked"

	tests := []struct {
		name          string
		newStatus     string
		initialQuests []Quest
		checkResult   func(t *testing.T, repo *mockQuestRepository)
		expectError   bool
	}{
		{
			name:      "completed with no remaining active transfers → quest completed",
			newStatus: "completed",
			initialQuests: []Quest{
				{ID: questID, Status: "in_progress", Transfers: []QuestTransfer{{TransferID: transferID, Status: "completed"}}, SourceRows: []int{1}},
			},
			checkResult: func(t *testing.T, repo *mockQuestRepository) {
				assert.Equal(t, "completed", repo.quests[0].Status)
			},
		},
		{
			name:      "completed with another active transfer → quest stays in_progress",
			newStatus: "completed",
			initialQuests: []Quest{
				{ID: questID, Status: "in_progress", Transfers: []QuestTransfer{
					{TransferID: transferID, Status: "completed"},
					{TransferID: 99, Status: "in_transit"},
				}, SourceRows: []int{1}},
			},
			checkResult: func(t *testing.T, repo *mockQuestRepository) {
				assert.Equal(t, "in_progress", repo.quests[0].Status)
			},
		},
		{
			name:      "cancelled → transfer removed, no active transfers left → quest pending",
			newStatus: "cancelled",
			initialQuests: []Quest{
				{ID: questID, Status: "in_progress", Transfers: []QuestTransfer{{TransferID: transferID, Status: "in_transit"}}, SourceRows: []int{1}},
			},
			checkResult: func(t *testing.T, repo *mockQuestRepository) {
				assert.Equal(t, "pending", repo.quests[0].Status)
				assert.Empty(t, repo.quests[0].Transfers)
			},
		},
		{
			name:      "cancelled → transfer removed, another active transfer remains → quest stays in_progress",
			newStatus: "cancelled",
			initialQuests: []Quest{
				{ID: questID, Status: "in_progress", Transfers: []QuestTransfer{
					{TransferID: transferID, Status: "in_transit"},
					{TransferID: 99, Status: "in_transit"},
				}, SourceRows: []int{1}},
			},
			checkResult: func(t *testing.T, repo *mockQuestRepository) {
				assert.Equal(t, "in_progress", repo.quests[0].Status)
				assert.Len(t, repo.quests[0].Transfers, 1)
				assert.Equal(t, 99, repo.quests[0].Transfers[0].TransferID)
			},
		},
		{
			name:          "no quest linked to transfer → no error",
			newStatus:     "completed",
			initialQuests: []Quest{},
			checkResult:   func(t *testing.T, repo *mockQuestRepository) {},
		},
		{
			name:      "unknown status → no action taken",
			newStatus: "unknown_status",
			initialQuests: []Quest{
				{ID: questID, Status: "in_progress", Transfers: []QuestTransfer{{TransferID: transferID, Status: "in_transit"}}, SourceRows: []int{1}},
			},
			checkResult: func(t *testing.T, repo *mockQuestRepository) {
				assert.Equal(t, "in_progress", repo.quests[0].Status)
				assert.Len(t, repo.quests[0].Transfers, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockQuestRepository{
				quests:   tt.initialQuests,
				mappings: make(map[string]int),
			}
			svc := &Service{questRepo: mockRepo}

			err := svc.OnTransferStatusChanged(transferID, tt.newStatus)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			tt.checkResult(t, mockRepo)
		})
	}
}

func TestHandler_ListCategoryMappings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mockRepo := setupTestHandler()

	mockRepo.categoryMappings = []CategoryMapping{
		{ID: 1, FormItemName: "Laptop Dell", CategoryID: 10, UseCount: 5},
		{ID: 2, FormItemName: "Mouse", CategoryID: 20, UseCount: 1},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/equipment-requests/category-mappings", nil)

	handler.ListCategoryMappings(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.EqualValues(t, 2, resp["count"])
	mappings, ok := resp["mappings"].([]interface{})
	require.True(t, ok)
	assert.Len(t, mappings, 2)
}

func TestHandler_DeleteCategoryMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		id             string
		initialData    []CategoryMapping
		expectedStatus int
	}{
		{
			name:           "Delete existing mapping returns 204",
			id:             "1",
			initialData:    []CategoryMapping{{ID: 1, FormItemName: "Laptop", CategoryID: 10}},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Delete non-existent mapping returns 404",
			id:             "99",
			initialData:    []CategoryMapping{},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid ID returns 400",
			id:             "abc",
			initialData:    []CategoryMapping{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockRepo := setupTestHandler()
			mockRepo.categoryMappings = tt.initialData

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "id", Value: tt.id}}
			c.Request = httptest.NewRequest("DELETE", "/api/equipment-requests/category-mappings/"+tt.id, nil)

			handler.DeleteCategoryMapping(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_GetSyncStatus_NoScheduler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := setupTestHandler() // scheduler is nil by default

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/equipment-requests/sync-status", nil)

	handler.GetSyncStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp SyncStatus
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp.Enabled)
	assert.Nil(t, resp.LastSync)
}

// compile-time check: mockTransferCreator implements TransferCreator
var _ TransferCreator = (*mockTransferCreator)(nil)
