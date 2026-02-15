package equipment_requests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockQuestRepository is a mock implementation for testing
type mockQuestRepository struct {
	quests       []Quest
	syncLogs     []SyncLog
	mappings     map[string]int
	statusUpdate func(ctx context.Context, questID string, status string) error
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
		if filter.Status == "" || q.Status == filter.Status {
			result = append(result, q)
		}
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
	// No-op for mock
	return nil
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
