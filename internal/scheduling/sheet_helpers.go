package scheduling

import (
	"fmt"
	"strconv"
	"strings"
)

// Column names expected in the spreadsheet header row.
// Change these constants if the spreadsheet layout changes.
const (
	colNickname      = "Pseudonim"
	colCity          = "Miasto"
	colHours         = "Godzin dyżuru"
	colAvailableFrom = "Dostępność od"
	colAvailableTo   = "Dostępność do"
	colNotes         = "Uwagi"
	colTags          = "Tagi"
)

// Tags that cause a volunteer row to be skipped during import.
var skipTags = []string{"Przegżdacz"}

// columnMap maps known header names to their column index in the sheet.
type columnMap struct {
	nickname      int
	city          int
	hours         int
	availableFrom int
	availableTo   int
	notes         int
	tags          int
}

// parseHeader reads the first row and builds a column map.
// Returns error if any required column is missing.
func parseHeader(headerRow []interface{}) (columnMap, error) {
	cm := columnMap{
		nickname: -1, city: -1, hours: -1,
		availableFrom: -1, availableTo: -1,
		notes: -1, tags: -1,
	}

	for i, cell := range headerRow {
		name := strings.TrimSpace(fmt.Sprintf("%v", cell))
		switch name {
		case colNickname:
			cm.nickname = i
		case colCity:
			cm.city = i
		case colHours:
			cm.hours = i
		case colAvailableFrom:
			cm.availableFrom = i
		case colAvailableTo:
			cm.availableTo = i
		case colNotes:
			cm.notes = i
		case colTags:
			cm.tags = i
		}
	}

	// Required columns
	missing := []string{}
	if cm.nickname < 0 {
		missing = append(missing, colNickname)
	}
	if cm.hours < 0 {
		missing = append(missing, colHours)
	}
	if cm.availableFrom < 0 {
		missing = append(missing, colAvailableFrom)
	}
	if cm.availableTo < 0 {
		missing = append(missing, colAvailableTo)
	}
	if len(missing) > 0 {
		return cm, fmt.Errorf("missing required columns: %s", strings.Join(missing, ", "))
	}

	return cm, nil
}

// shouldSkipByTag returns true if the row's tag column contains a skip tag.
func shouldSkipByTag(row []interface{}, tagIdx int) bool {
	if tagIdx < 0 {
		return false
	}
	tag := cellStr(row, tagIdx)
	if tag == "" {
		return false
	}
	for _, skip := range skipTags {
		if strings.EqualFold(tag, skip) {
			return true
		}
	}
	return false
}

func cellStr(row []interface{}, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", row[idx]))
}

func cellStrPtr(row []interface{}, idx int) *string {
	s := cellStr(row, idx)
	if s == "" {
		return nil
	}
	return &s
}

func cellInt(row []interface{}, idx int, defaultVal int) int {
	s := cellStr(row, idx)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
