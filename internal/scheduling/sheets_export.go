package scheduling

import (
	"fmt"
	"warehouse/internal/integrations/googlesheets"
)

// ExportToSheet writes the schedule data to a Google Spreadsheet.
// It clears the target sheet first, then writes all rows starting at A1.
func ExportToSheet(
	sheetsHandler *googlesheets.GoogleSheetsHandler,
	sheetID, sheetName string,
	slots []Slot,
	volunteers []Volunteer,
	assignments []Assignment,
) (int, error) {
	rows := buildExportRows(slots, volunteers, assignments)

	// Convert [][]string → [][]interface{} for Google Sheets API
	sheetRows := make([][]interface{}, len(rows))
	for i, row := range rows {
		iRow := make([]interface{}, len(row))
		for j, cell := range row {
			iRow[j] = cell
		}
		sheetRows[i] = iRow
	}

	// Clear the sheet
	if err := sheetsHandler.ClearSheet(sheetID, sheetName); err != nil {
		return 0, fmt.Errorf("failed to clear sheet: %w", err)
	}

	// Write data starting at A1
	writeRange := fmt.Sprintf("%s!A1", sheetName)
	if err := sheetsHandler.WriteSpreadsheet(sheetID, writeRange, sheetRows); err != nil {
		return 0, fmt.Errorf("failed to write to sheet: %w", err)
	}

	return len(sheetRows), nil
}
