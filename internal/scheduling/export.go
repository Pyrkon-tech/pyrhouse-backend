package scheduling

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ExportCSV generates a CSV schedule with hourly granularity for festival days
// and single rows for montage/demontage days.
// Columns: Godzina, Stanowisko 1, Stanowisko 2, ..., Stanowisko N
func ExportCSV(schedule *Schedule, slots []Slot, volunteers []Volunteer, assignments []Assignment) string {
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
		// Section header
		rows = append(rows, sectionRow(fmt.Sprintf("--- %s ---", label), maxCols))
		// Single row with all assigned volunteers
		row := makeRow(label, slotVolunteers[s.ID], maxCols)
		rows = append(rows, row)
	}

	// Festival: hourly breakdown
	if len(festivalSlots) > 0 {
		// Group by date for section headers
		currentDate := ""
		festStart := festivalSlots[0].StartTime
		festEnd := festivalSlots[len(festivalSlots)-1].EndTime

		for t := festStart; t.Before(festEnd); t = t.Add(time.Hour) {
			dateStr := t.Format("2006-01-02")
			if dateStr != currentDate {
				currentDate = dateStr
				dayName := polishWeekday(t.Weekday())
				rows = append(rows, sectionRow(fmt.Sprintf("--- %s ---", dayName), maxCols))
			}

			// Find which volunteers are on duty at this hour
			var onDuty []string
			for _, s := range festivalSlots {
				if !t.Before(s.StartTime) && t.Before(s.EndTime) {
					onDuty = append(onDuty, slotVolunteers[s.ID]...)
				}
			}

			hourLabel := fmt.Sprintf("%02d:00-%02d:00", t.Hour(), t.Add(time.Hour).Hour())
			rows = append(rows, makeRow(hourLabel, onDuty, maxCols))
		}
	}

	// Demontage days
	for _, s := range demontageSlots {
		label := ""
		if s.Label != nil {
			label = *s.Label
		}
		rows = append(rows, sectionRow(fmt.Sprintf("--- %s ---", label), maxCols))
		row := makeRow(label, slotVolunteers[s.ID], maxCols)
		rows = append(rows, row)
	}

	// Convert to CSV string
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
