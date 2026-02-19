package items

import (
	"encoding/json"
	"testing"
	"time"
	"warehouse/internal/models"

	"github.com/stretchr/testify/assert"
)

// Tests for JSON serialization of response wrappers — confirming frontend contract:
// asset response has "assetLogs", stock response has "logs"

func TestAssetWithLogs_JSON_HasAssetLogsField(t *testing.T) {
	asset := &models.Asset{ID: 1}
	logs := []models.AuditLog{
		{
			ID:           10,
			ResourceID:   1,
			ResourceType: "asset",
			Action:       "create",
			Data:         map[string]interface{}{"msg": "Asset created"},
			CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	resp := &assetWithLogs{Asset: asset, AssetLogs: logs}

	raw, err := json.Marshal(resp)
	assert.NoError(t, err)

	var m map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw, &m))

	assert.Contains(t, m, "assetLogs", "response must contain assetLogs key")
	assert.NotContains(t, m, "logs", "asset response must not contain logs key")

	entries, ok := m["assetLogs"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, entries, 1)

	entry := entries[0].(map[string]interface{})
	assert.Equal(t, "create", entry["action"])
}

func TestAssetWithLogs_JSON_EmptyLogsIsArray(t *testing.T) {
	resp := &assetWithLogs{Asset: &models.Asset{ID: 2}, AssetLogs: []models.AuditLog{}}

	raw, err := json.Marshal(resp)
	assert.NoError(t, err)

	var m map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw, &m))

	entries, ok := m["assetLogs"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, entries, 0)
}

func TestStockWithLogs_JSON_HasLogsField(t *testing.T) {
	stock := &models.StockItem{ID: 5, Quantity: 10}
	logs := []models.AuditLog{
		{
			ID:           20,
			ResourceID:   5,
			ResourceType: "stock",
			Action:       "update",
			Data:         map[string]interface{}{"msg": "Quantity updated", "quantity": float64(10)},
			CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	resp := &stockWithLogs{StockItem: stock, Logs: logs}

	raw, err := json.Marshal(resp)
	assert.NoError(t, err)

	var m map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw, &m))

	assert.Contains(t, m, "logs", "response must contain logs key")
	assert.NotContains(t, m, "assetLogs", "stock response must not contain assetLogs key")

	entries, ok := m["logs"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, entries, 1)
}

func TestStockWithLogs_JSON_EmptyLogsIsArray(t *testing.T) {
	resp := &stockWithLogs{StockItem: &models.StockItem{ID: 3}, Logs: []models.AuditLog{}}

	raw, err := json.Marshal(resp)
	assert.NoError(t, err)

	var m map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw, &m))

	entries, ok := m["logs"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, entries, 0)
}

// Table-driven: fetchItem returns error for unsupported category type
func TestFetchItem_UnsupportedCategory(t *testing.T) {
	tests := []struct {
		name     string
		category string
	}{
		{"empty category", ""},
		{"unknown category", "widget"},
		{"typo", "assets"},
	}

	svc := &ItemService{} // repos are nil — fetchItem bails before using them for unknown type

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id := 1
			_, err := svc.fetchItem(RetrieveItemQuery{ID: &id, CategoryType: tc.category})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported category type")
		})
	}
}
