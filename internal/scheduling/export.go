package scheduling

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// buildExportRows generates the schedule data as a 2D string grid.
// Used by both CSV export and Google Sheets export.
func buildExportRows(slots []Slot, volunteers []Volunteer, assignments []Assignment) [][]string {
	// Build lookup maps
	volMap := make(map[int]string)
	for _, v := range volunteers {
		volMap[v.ID] = v.Nickname
	}

	// slot ID → list of volunteer nicknames
	slotVolunteers := make(map[int][]string)
	for _, a := range assignments {
		if nick, ok := volMap[a.VolunteerID]; ok {
			slotVolunteers[a.SlotID] = append(slotVolunteers[a.SlotID], nick)
		}
	}

	// Find max capacity across all slots (determines number of columns)
	maxCols := 0
	for _, s := range slots {
		if s.Capacity > maxCols {
			maxCols = s.Capacity
		}
		if len(slotVolunteers[s.ID]) > maxCols {
			maxCols = len(slotVolunteers[s.ID])
		}
	}
	if maxCols == 0 {
		maxCols = 4
	}

	// Build header
	header := make([]string, maxCols+1)
	header[0] = "Godzina"
	for i := 1; i <= maxCols; i++ {
		header[i] = fmt.Sprintf("Stanowisko %d", i)
	}

	var rows [][]string
	rows = append(rows, header)

	// Group slots by type and sort
	var montageSlots, festivalSlots, demontageSlots []Slot
	for _, s := range slots {
		switch s.SlotType {
		case SlotTypeMontage:
			montageSlots = append(montageSlots, s)
		case SlotTypeFestival:
			festivalSlots = append(festivalSlots, s)
		case SlotTypeDemontage:
			demontageSlots = append(demontageSlots, s)
		}
	}

	sortByStart := func(s []Slot) {
		sort.Slice(s, func(i, j int) bool {
			return s[i].StartTime.Before(s[j].StartTime)
		})
	}
	sortByStart(montageSlots)
	sortByStart(festivalSlots)
	sortByStart(demontageSlots)

	// Montage days
	for _, s := range montageSlots {
		label := ""
		if s.Label != nil {
			label = *s.Label
		}
		rows = append(rows, sectionRow(fmt.Sprintf("--- %s ---", label), maxCols))
		rows = append(rows, makeRow(label, slotVolunteers[s.ID], maxCols))
	}

	// Festival: hourly breakdown
	if len(festivalSlots) > 0 {
		// Index volunteers per hour: hour truncated to hour → nicknames
		type hourKey = time.Time
		hourVolunteers := make(map[hourKey][]string)
		for _, s := range festivalSlots {
			for h := s.StartTime.Truncate(time.Hour); h.Before(s.EndTime); h = h.Add(time.Hour) {
				hourVolunteers[h] = append(hourVolunteers[h], slotVolunteers[s.ID]...)
			}
		}

		currentDate := ""
		festStart := festivalSlots[0].StartTime.Truncate(time.Hour)
		festEnd := festivalSlots[len(festivalSlots)-1].EndTime

		for t := festStart; t.Before(festEnd); t = t.Add(time.Hour) {
			dateStr := t.Format("2006-01-02")
			if dateStr != currentDate {
				currentDate = dateStr
				rows = append(rows, sectionRow(fmt.Sprintf("--- %s ---", polishWeekday(t.Weekday())), maxCols))
			}
			hourLabel := fmt.Sprintf("%02d:00-%02d:00", t.Hour(), t.Add(time.Hour).Hour())
			rows = append(rows, makeRow(hourLabel, hourVolunteers[t], maxCols))
		}
	}

	// Demontage days
	for _, s := range demontageSlots {
		label := ""
		if s.Label != nil {
			label = *s.Label
		}
		rows = append(rows, sectionRow(fmt.Sprintf("--- %s ---", label), maxCols))
		rows = append(rows, makeRow(label, slotVolunteers[s.ID], maxCols))
	}

	return rows
}

// ExportCSV generates a CSV string from the schedule data.
func ExportCSV(schedule *Schedule, slots []Slot, volunteers []Volunteer, assignments []Assignment) string {
	rows := buildExportRows(slots, volunteers, assignments)

	var lines []string
	for _, row := range rows {
		lines = append(lines, strings.Join(row, ","))
	}

	return strings.Join(lines, "\n")
}

func sectionRow(title string, maxCols int) []string {
	row := make([]string, maxCols+1)
	row[0] = title
	return row
}

func makeRow(label string, names []string, maxCols int) []string {
	row := make([]string, maxCols+1)
	row[0] = label
	for i, name := range names {
		if i < maxCols {
			row[i+1] = name
		}
	}
	return row
}
