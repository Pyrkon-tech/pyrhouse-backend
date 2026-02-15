package equipment_requests

import (
	"testing"

	"warehouse/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestService_questKey(t *testing.T) {
	service := &Service{}

	tests := []struct {
		name     string
		row      SheetRow
		expected string
	}{
		{
			name: "Full quest key",
			row: SheetRow{
				Pavilion:     "PCC",
				Location:     "Maskarada",
				Recipient:    "Jan Kowalski",
				DeliveryDate: "2025-06-13",
				PickupTime:   "17-18",
			},
			expected: "PCC|Maskarada|Jan Kowalski|2025-06-13|17-18",
		},
		{
			name: "Quest key without pickup time",
			row: SheetRow{
				Pavilion:     "Pawilon 5",
				Location:     "POW",
				Recipient:    "Anna Nowak",
				DeliveryDate: "2025-06-14",
				PickupTime:   "",
			},
			expected: "Pawilon 5|POW|Anna Nowak|2025-06-14|",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.questKey(tt.row)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestService_aggregateQuests(t *testing.T) {
	service := &Service{}

	tests := []struct {
		name          string
		rows          []SheetRow
		expectedCount int
	}{
		{
			name: "Single quest with multiple items",
			rows: []SheetRow{
				{
					RowNumber:    1,
					Item:         "Laptop",
					Quantity:     2,
					Pavilion:     "PCC",
					Location:     "Maskarada",
					Recipient:    "Jan Kowalski",
					DeliveryDate: "2025-06-13",
					Status:       StatusOrdered,
				},
				{
					RowNumber:    2,
					Item:         "Mouse",
					Quantity:     2,
					Pavilion:     "PCC",
					Location:     "Maskarada",
					Recipient:    "Jan Kowalski",
					DeliveryDate: "2025-06-13",
					Status:       StatusOrdered,
				},
			},
			expectedCount: 1, // Same destination/recipient/date = 1 quest
		},
		{
			name: "Multiple quests - different recipients",
			rows: []SheetRow{
				{
					RowNumber:    1,
					Item:         "Laptop",
					Quantity:     1,
					Pavilion:     "PCC",
					Location:     "Maskarada",
					Recipient:    "Jan Kowalski",
					DeliveryDate: "2025-06-13",
					Status:       StatusOrdered,
				},
				{
					RowNumber:    2,
					Item:         "Mouse",
					Quantity:     1,
					Pavilion:     "PCC",
					Location:     "Maskarada",
					Recipient:    "Anna Nowak", // Different recipient
					DeliveryDate: "2025-06-13",
					Status:       StatusOrdered,
				},
			},
			expectedCount: 2, // Different recipients = 2 quests
		},
		{
			name: "Filter out non-ordered items",
			rows: []SheetRow{
				{
					RowNumber:    1,
					Item:         "Laptop",
					Quantity:     1,
					Pavilion:     "PCC",
					Location:     "Maskarada",
					Recipient:    "Jan Kowalski",
					DeliveryDate: "2025-06-13",
					Status:       StatusOrdered,
				},
				{
					RowNumber:    2,
					Item:         "Mouse",
					Quantity:     1,
					Pavilion:     "PCC",
					Location:     "Maskarada",
					Recipient:    "Jan Kowalski",
					DeliveryDate: "2025-06-13",
					Status:       StatusDelivered, // Not ordered
				},
			},
			expectedCount: 1, // Only 1 ordered item = 1 quest with 1 item
		},
		{
			name: "Different pickup times = different quests",
			rows: []SheetRow{
				{
					RowNumber:    1,
					Item:         "Laptop",
					Quantity:     1,
					Pavilion:     "PCC",
					Location:     "Maskarada",
					Recipient:    "Jan Kowalski",
					DeliveryDate: "2025-06-13",
					PickupTime:   "17-18",
					Status:       StatusOrdered,
				},
				{
					RowNumber:    2,
					Item:         "Mouse",
					Quantity:     1,
					Pavilion:     "PCC",
					Location:     "Maskarada",
					Recipient:    "Jan Kowalski",
					DeliveryDate: "2025-06-13",
					PickupTime:   "19-20", // Different pickup time
					Status:       StatusOrdered,
				},
			},
			expectedCount: 2, // Different pickup times = 2 quests
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quests := service.aggregateQuests(tt.rows)
			assert.Equal(t, tt.expectedCount, len(quests))

			// Verify quest structure
			for _, quest := range quests {
				assert.NotEmpty(t, quest.ID)
				assert.NotEmpty(t, quest.Destination.Pavilion)
				assert.NotEmpty(t, quest.Destination.Location)
				assert.NotEmpty(t, quest.Recipient)
				assert.NotEmpty(t, quest.Items)
				assert.Equal(t, "pending", quest.Status)
				assert.NotEmpty(t, quest.SourceRows)
			}
		})
	}
}

func TestService_aggregateQuests_ItemsGrouping(t *testing.T) {
	service := &Service{}

	rows := []SheetRow{
		{
			RowNumber:    10,
			Item:         "Laptop",
			Quantity:     2,
			Pavilion:     "PCC",
			Location:     "Maskarada",
			Recipient:    "Jan Kowalski",
			DeliveryDate: "2025-06-13",
			PickupTime:   "17-18",
			BudgetOwner:  "Jan Kowalski",
			Notes:        "Test note 1",
			Status:       StatusOrdered,
		},
		{
			RowNumber:    11,
			Item:         "Mouse",
			Quantity:     3,
			Pavilion:     "PCC",
			Location:     "Maskarada",
			Recipient:    "Jan Kowalski",
			DeliveryDate: "2025-06-13",
			PickupTime:   "17-18",
			BudgetOwner:  "Jan Kowalski",
			Notes:        "Test note 2",
			Status:       StatusOrdered,
		},
	}

	quests := service.aggregateQuests(rows)

	assert.Equal(t, 1, len(quests))
	quest := quests[0]

	// Verify quest has both items
	assert.Equal(t, 2, len(quest.Items))
	assert.Equal(t, []int{10, 11}, quest.SourceRows)

	// Verify first item
	assert.Equal(t, "Laptop", quest.Items[0].Name)
	assert.Equal(t, 2, quest.Items[0].Quantity)
	assert.Equal(t, "Test note 1", quest.Items[0].Notes)

	// Verify second item
	assert.Equal(t, "Mouse", quest.Items[1].Name)
	assert.Equal(t, 3, quest.Items[1].Quantity)
	assert.Equal(t, "Test note 2", quest.Items[1].Notes)

	// Verify quest metadata
	assert.Equal(t, "PCC", quest.Destination.Pavilion)
	assert.Equal(t, "Maskarada", quest.Destination.Location)
	assert.Equal(t, "Jan Kowalski", quest.Recipient)
	assert.Equal(t, "2025-06-13", quest.DeliveryDate)
	assert.Equal(t, "17-18", quest.PickupTime)
	assert.Equal(t, "Jan Kowalski", quest.BudgetOwner)
}

func TestGenerateQuestID(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{
			name: "Standard quest key",
			key:  "PCC|Maskarada|Jan Kowalski|2025-06-13|17-18",
		},
		{
			name: "Quest key without pickup time",
			key:  "PCC|Maskarada|Jan Kowalski|2025-06-13|",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := generateQuestID(tt.key)

			// Verify format
			assert.True(t, len(id) > 0)
			assert.Contains(t, id, "quest-")

			// Verify deterministic (same key = same ID)
			id2 := generateQuestID(tt.key)
			assert.Equal(t, id, id2)

			// Verify different keys = different IDs
			differentID := generateQuestID(tt.key + "different")
			assert.NotEqual(t, id, differentID)
		})
	}
}

func TestService_matchCategory(t *testing.T) {
	// This test requires mocking the category repository
	// For now, test the basic "none" case with empty categories
	service := &Service{
		categories: []models.ItemCategory{},
	}

	result := service.matchCategory("Laptop")

	assert.Equal(t, "none", result.MatchType)
	assert.Equal(t, 0.0, result.Confidence)
	assert.Nil(t, result.CategoryID)
}
