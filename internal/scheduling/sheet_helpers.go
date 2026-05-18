package scheduling

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Column names expected in the spreadsheet header row.
// Change these constants if the spreadsheet layout changes.
const (
	colNickname         = "Pseudonim"
	colCity             = "Miasto"
	colHours            = "Deklaracja godzin dyżuru"
	colAvailableFrom    = "Przyjazd"
	colAvailableTo      = "Wyjazd"
	colNotes            = "Uwagi"
	colTags             = "Tagi"
	colDiscordConfirmed = "Discord (potwierdzony)"
)

// Tags that cause a volunteer row to be skipped during import.
var skipTags = []string{"Przegżdacz"}

// columnMap maps known header names to their column index in the sheet.
type columnMap struct {
	nickname         int
	city             int
	hours            int
	availableFrom    int
	availableTo      int
	notes            int
	tags             int
	discordConfirmed int
}

// parseHeader reads the first row and builds a column map.
// Returns error if any required column is missing.
func parseHeader(headerRow []interface{}) (columnMap, error) {
	cm := columnMap{
		nickname: -1, city: -1, hours: -1,
		availableFrom: -1, availableTo: -1,
		notes: -1, tags: -1, discordConfirmed: -1,
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
		case colDiscordConfirmed:
			cm.discordConfirmed = i
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

// polishDayToWeekday maps lowercase Polish day names to time.Weekday.
var polishDayToWeekday = map[string]time.Weekday{
	"poniedziałek": time.Monday,
	"wtorek":       time.Tuesday,
	"środa":        time.Wednesday,
	"czwartek":     time.Thursday,
	"piątek":       time.Friday,
	"sobota":       time.Saturday,
	"niedziela":    time.Sunday,
}

// warsawLocation is loaded once at startup; falls back to UTC on error.
var warsawLocation = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// parseAvailabilityTime parses a date/time string from the spreadsheet.
// Supported formats:
//   - "2026-06-18 14:00:00" or "2026-06-18 14:00" (treated as Europe/Warsaw local time)
//   - "wtorek, 08:00" (Polish weekday + time, resolved within event range)
func parsePolishDayTime(raw string, eventStart, eventEnd time.Time) (time.Time, error) {
	// Try ISO datetime formats first — parse as Warsaw local time
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04"} {
		if t, err := time.ParseInLocation(layout, raw, warsawLocation); err == nil {
			return t, nil
		}
	}

	// Fall back to "dzień, HH:MM"
	parts := strings.SplitN(raw, ",", 2)
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("expected format 'dzień, HH:MM', got %q", raw)
	}

	dayName := strings.TrimSpace(strings.ToLower(parts[0]))
	timeStr := strings.TrimSpace(parts[1])

	weekday, ok := polishDayToWeekday[dayName]
	if !ok {
		return time.Time{}, fmt.Errorf("unknown day name %q", parts[0])
	}

	timeParts := strings.SplitN(timeStr, ":", 2)
	if len(timeParts) != 2 {
		return time.Time{}, fmt.Errorf("invalid time format %q", timeStr)
	}
	hour, err := strconv.Atoi(timeParts[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid hour in %q", timeStr)
	}
	minute, err := strconv.Atoi(timeParts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid minute in %q", timeStr)
	}

	for d := eventStart; !d.After(eventEnd); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == weekday {
			return time.Date(d.Year(), d.Month(), d.Day(), hour, minute, 0, 0, d.Location()), nil
		}
	}

	return time.Time{}, fmt.Errorf("day %q not found within event range %s – %s",
		parts[0], eventStart.Format("2006-01-02"), eventEnd.Format("2006-01-02"))
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
