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
	sr.PickupTime = normalizePickupTime(getCell("pickup_time"))
	sr.DeliveryDate = getCell("delivery_date")
	sr.BudgetOwner = getCell("budget_owner")
	sr.Recipient = getCell("recipient")
	sr.Notes = getCell("notes")

	// Parse quantity — optional. A blank, zero, negative or unparseable value is treated as
	// "not specified" (nil) so the row is still imported and surfaced for the dispatcher to fix,
	// instead of being silently dropped.
	if qtyStr := getCell("quantity"); qtyStr != "" {
		if qty, err := strconv.Atoi(qtyStr); err == nil && qty > 0 {
			sr.Quantity = &qty
		}
	}

	// A row only needs an item name; the quantity may be filled in later.
	if sr.Item == "" {
		return nil, fmt.Errorf("row %d missing required field (item)", rowNumber)
	}

	return sr, nil
}

// normalizePickupTime canonicalises pickup-time strings to "HH:MM" so that cosmetic
// formatting differences in the sheet (e.g. "10.00", "10:00:00", "9:00") do not change
// the quest_key and spawn duplicate quests. Values that are not recognisable times are
// returned trimmed but otherwise unchanged.
func normalizePickupTime(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}

	// Split on the time separators used in the sheet (":" or ".").
	fields := strings.FieldsFunc(s, func(r rune) bool { return r == ':' || r == '.' })
	if len(fields) < 2 {
		return s
	}

	hour, err1 := strconv.Atoi(strings.TrimSpace(fields[0]))
	minute, err2 := strconv.Atoi(strings.TrimSpace(fields[1]))
	if err1 != nil || err2 != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return s
	}

	return fmt.Sprintf("%02d:%02d", hour, minute)
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
