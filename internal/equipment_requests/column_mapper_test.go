package equipment_requests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// intPtr is a test helper for building optional-int fields (e.g. QuestItem.Quantity).
func intPtr(i int) *int { return &i }

func TestNormalizePickupTime(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty stays empty", "", ""},
		{"whitespace trimmed to empty", "   ", ""},
		{"dot separator", "10.00", "10:00"},
		{"seconds dropped", "10:00:00", "10:00"},
		{"already canonical", "10:00", "10:00"},
		{"single-digit hour padded", "9:00", "09:00"},
		{"dot with seconds", "16.00.00", "16:00"},
		{"surrounding spaces", "  12:30:00 ", "12:30"},
		{"non-time left as-is", "rano", "rano"},
		{"bare hour left as-is", "10", "10"},
		{"out-of-range left as-is", "25:00", "25:00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizePickupTime(tt.in))
		})
	}
}

func TestNewColumnMapper(t *testing.T) {
	headers := []string{
		"Rzeczy",
		"Ilość",
		"Pawilon",
		"Miejsce",
		"Stan",
		"Godzina odbioru",
		"Dostawa do",
		"Osoba odpowiedzialna za budżet",
		"Do kogo ma trafić",
		"UWAGI",
	}

	mapper := NewColumnMapper(headers)

	// Verify all fields are mapped
	assert.NotNil(t, mapper.headerIndex)
	assert.Equal(t, 10, len(mapper.headerIndex))
}

func TestColumnMapper_ParseRow(t *testing.T) {
	headers := []string{
		"Rzeczy",
		"Ilość",
		"Pawilon",
		"Miejsce",
		"Stan",
		"Godzina odbioru",
		"Dostawa do",
		"Osoba odpowiedzialna za budżet",
		"Do kogo ma trafić",
		"UWAGI",
	}

	tests := []struct {
		name      string
		row       []string
		rowNumber int
		expected  *SheetRow
		wantErr   bool
	}{
		{
			name: "Valid row",
			row: []string{
				"Laptop",       // Rzeczy
				"4",            // Ilość
				"PCC",          // Pawilon
				"Maskarada",    // Miejsce
				"Zamówione",    // Stan
				"17-18",        // Godzina odbioru
				"2025-06-13",   // Dostawa do
				"Jan Kowalski", // Osoba odpowiedzialna za budżet
				"Anna Nowak",   // Do kogo ma trafić
				"Test note",    // UWAGI
			},
			rowNumber: 5,
			expected: &SheetRow{
				RowNumber:    5,
				Item:         "Laptop",
				Quantity:     intPtr(4),
				Pavilion:     "PCC",
				Location:     "Maskarada",
				Status:       "Zamówione",
				PickupTime:   "17-18",
				DeliveryDate: "2025-06-13",
				BudgetOwner:  "Jan Kowalski",
				Recipient:    "Anna Nowak",
				Notes:        "Test note",
			},
			wantErr: false,
		},
		{
			name: "Missing item",
			row: []string{
				"",          // Rzeczy (empty)
				"4",         // Ilość
				"PCC",       // Pawilon
				"Maskarada", // Miejsce
				"Zamówione", // Stan
			},
			rowNumber: 10,
			expected:  nil,
			wantErr:   true,
		},
		{
			name: "Unparseable quantity is kept as unspecified",
			row: []string{
				"Laptop", // Rzeczy
				"abc",    // Ilość (invalid → nil)
				"PCC",    // Pawilon
			},
			rowNumber: 15,
			expected: &SheetRow{
				RowNumber: 15,
				Item:      "Laptop",
				Quantity:  nil,
				Pavilion:  "PCC",
			},
			wantErr: false,
		},
		{
			name: "Blank quantity is kept as unspecified",
			row: []string{
				"Laptop", // Rzeczy
				"",       // Ilość (blank → nil)
				"PCC",    // Pawilon
			},
			rowNumber: 20,
			expected: &SheetRow{
				RowNumber: 20,
				Item:      "Laptop",
				Quantity:  nil,
				Pavilion:  "PCC",
			},
			wantErr: false,
		},
	}

	mapper := NewColumnMapper(headers)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mapper.ParseRow(tt.row, tt.rowNumber)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.RowNumber, result.RowNumber)
				assert.Equal(t, tt.expected.Item, result.Item)
				assert.Equal(t, tt.expected.Quantity, result.Quantity)
				assert.Equal(t, tt.expected.Pavilion, result.Pavilion)
				assert.Equal(t, tt.expected.Location, result.Location)
				assert.Equal(t, tt.expected.Status, result.Status)
				assert.Equal(t, tt.expected.PickupTime, result.PickupTime)
				assert.Equal(t, tt.expected.DeliveryDate, result.DeliveryDate)
				assert.Equal(t, tt.expected.BudgetOwner, result.BudgetOwner)
				assert.Equal(t, tt.expected.Recipient, result.Recipient)
				assert.Equal(t, tt.expected.Notes, result.Notes)
			}
		})
	}
}

func TestColumnMapper_HasRequiredColumns(t *testing.T) {
	tests := []struct {
		name     string
		headers  []string
		expected bool
	}{
		{
			name: "All required columns present",
			headers: []string{
				"Rzeczy",
				"Ilość",
				"Pawilon",
				"Miejsce",
				"Do kogo ma trafić",
				"Dostawa do",
			},
			expected: true,
		},
		{
			name: "Missing required column",
			headers: []string{
				"Rzeczy",
				"Ilość",
				"Pawilon",
				// Missing "Miejsce"
				"Do kogo ma trafić",
				"Dostawa do",
			},
			expected: false,
		},
		{
			name:     "Empty headers",
			headers:  []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewColumnMapper(tt.headers)
			result := mapper.HasRequiredColumns()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestColumnMapper_MissingColumns(t *testing.T) {
	headers := []string{
		"Rzeczy",
		"Ilość",
		"Pawilon",
		// Missing "Miejsce", "Do kogo ma trafić", "Dostawa do"
	}

	mapper := NewColumnMapper(headers)
	missing := mapper.MissingColumns()

	assert.NotNil(t, missing)
	assert.Contains(t, missing, "Miejsce")
	assert.Contains(t, missing, "Do kogo ma trafić")
	assert.Contains(t, missing, "Dostawa do")
}
