package equipment_requests

import (
	"fmt"
	"strconv"
	"strings"
)

// ColumnMapping maps logical field names to Polish column headers
var ColumnMapping = map[string]string{
	"item":          "Rzeczy",
	"quantity":      "Ilość",
	"pavilion":      "Pawilon",
	"location":      "Miejsce",
	"status":        "Stan",
	"pickup_time":   "Godzina odbioru",
	"delivery_date": "Dostawa do",
	"budget_owner":  "Osoba odpowiedzialna za budżet",
	"recipient":     "Do kogo ma trafić",
	"notes":         "UWAGI",
}

// ColumnMapper handles flexible column mapping from spreadsheet
type ColumnMapper struct {
	headerIndex map[string]int
}

// NewColumnMapper creates a mapper from header row
func NewColumnMapper(headers []string) *ColumnMapper {
	index := make(map[string]int)

	// Build reverse lookup: column name -> index
	for i, header := range headers {
		header = strings.TrimSpace(header)
		for field, expectedName := range ColumnMapping {
			if header == expectedName {
				index[field] = i
				break
			}
		}
	}

	return &ColumnMapper{headerIndex: index}
}

// ParseRow converts spreadsheet row to SheetRow struct
func (m *ColumnMapper) ParseRow(row []string, rowNumber int) (*SheetRow, error) {
	sr := &SheetRow{RowNumber: rowNumber}

	// Helper to safely get cell value
	getCell := func(field string) string {
		if idx, ok := m.headerIndex[field]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	sr.Item = getCell("item")
	sr.Pavilion = getCell("pavilion")
	sr.Location = getCell("location")
	sr.Status = getCell("status")
	sr.PickupTime = getCell("pickup_time")
	sr.DeliveryDate = getCell("delivery_date")
	sr.BudgetOwner = getCell("budget_owner")
	sr.Recipient = getCell("recipient")
	sr.Notes = getCell("notes")

	// Parse quantity
	qtyStr := getCell("quantity")
	if qtyStr != "" {
		qty, err := strconv.Atoi(qtyStr)
		if err != nil {
			return nil, fmt.Errorf("invalid quantity '%s' in row %d: %w", qtyStr, rowNumber, err)
		}
		sr.Quantity = qty
	}

	// Validate required fields
	if sr.Item == "" || sr.Quantity == 0 {
		return nil, fmt.Errorf("row %d missing required fields (item or quantity)", rowNumber)
	}

	return sr, nil
}

// HasRequiredColumns checks if all required columns are present
func (m *ColumnMapper) HasRequiredColumns() bool {
	required := []string{"item", "quantity", "pavilion", "location", "recipient", "delivery_date"}
	for _, field := range required {
		if _, ok := m.headerIndex[field]; !ok {
			return false
		}
	}
	return true
}

// MissingColumns returns list of missing required columns
func (m *ColumnMapper) MissingColumns() []string {
	required := []string{"item", "quantity", "pavilion", "location", "recipient", "delivery_date"}
	var missing []string
	for _, field := range required {
		if _, ok := m.headerIndex[field]; !ok {
			missing = append(missing, ColumnMapping[field])
		}
	}
	return missing
}
