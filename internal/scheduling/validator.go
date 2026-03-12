package scheduling

import (
	"fmt"
	"sort"
	"time"
)

// Validate checks the schedule assignments against all constraints and returns issues found.
func Validate(slots []Slot, volunteers []Volunteer, assignments []Assignment) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Build lookup maps
	slotMap := make(map[int]Slot)
	for _, s := range slots {
		slotMap[s.ID] = s
	}

	volMap := make(map[int]Volunteer)
	for _, v := range volunteers {
		volMap[v.ID] = v
	}

	// Group assignments by volunteer and by slot
	volAssignments := make(map[int][]int)  // volunteer ID → slot IDs
	slotAssignments := make(map[int][]int) // slot ID → volunteer IDs
	for _, a := range assignments {
		volAssignments[a.VolunteerID] = append(volAssignments[a.VolunteerID], a.SlotID)
		slotAssignments[a.SlotID] = append(slotAssignments[a.SlotID], a.VolunteerID)
	}

	// Check each volunteer
	for _, v := range volunteers {
		slotIDs := volAssignments[v.ID]
		assignedSlots := make([]Slot, 0, len(slotIDs))
		var totalHours float64
		hasFestival := false

		for _, sid := range slotIDs {
			s := slotMap[sid]
			assignedSlots = append(assignedSlots, s)
			totalHours += s.CreditHours
			if s.SlotType == SlotTypeFestival {
				hasFestival = true
			}
		}

		// Under hours
		if totalHours < float64(v.TargetHours) {
			result.Valid = false
			result.Issues = append(result.Issues, ValidationIssue{
				Type:        "under_hours",
				Severity:    "warning",
				Volunteer:   v.Nickname,
				VolunteerID: &v.ID,
				Assigned:    int(totalHours),
				Target:      v.TargetHours,
				Message:     fmt.Sprintf("%s ma za mało godzin (%d/%dh)", v.Nickname, int(totalHours), v.TargetHours),
			})
		}

		// Over hours
		if totalHours > 18 {
			result.Issues = append(result.Issues, ValidationIssue{
				Type:        "over_hours",
				Severity:    "warning",
				Volunteer:   v.Nickname,
				VolunteerID: &v.ID,
				Assigned:    int(totalHours),
				Target:      v.TargetHours,
				Message:     fmt.Sprintf("%s ma za dużo godzin (%dh)", v.Nickname, int(totalHours)),
			})
		}

		// No festival shifts
		if !hasFestival && len(slotIDs) > 0 {
			result.Issues = append(result.Issues, ValidationIssue{
				Type:        "no_festival_shifts",
				Severity:    "warning",
				Volunteer:   v.Nickname,
				VolunteerID: &v.ID,
				Message:     fmt.Sprintf("%s nie ma żadnych dyżurów festiwalowych", v.Nickname),
			})
		}

		// Outside availability
		for _, s := range assignedSlots {
			if s.StartTime.Before(v.AvailableFrom) || s.EndTime.After(v.AvailableTo) {
				label := slotLabel(s)
				result.Issues = append(result.Issues, ValidationIssue{
					Type:        "outside_availability",
					Severity:    "warning",
					Volunteer:   v.Nickname,
					VolunteerID: &v.ID,
					Slot:        label,
					SlotID:      &s.ID,
					Message:     fmt.Sprintf("%s przypisany poza dostępnością: %s", v.Nickname, label),
				})
			}
		}

		// Double-booked (overlapping slots for same volunteer)
		checkDoubleBooked(assignedSlots, v, result)

		// Consecutive hours and break constraints
		checkContinuousAndBreaks(assignedSlots, v, result)
	}

	// Check slot staffing
	for _, s := range slots {
		assigned := len(slotAssignments[s.ID])
		label := slotLabel(s)

		if assigned < s.Capacity {
			result.Issues = append(result.Issues, ValidationIssue{
				Type:     "slot_understaffed",
				Severity: "warning",
				Slot:     label,
				SlotID:   &s.ID,
				Assigned: assigned,
				Capacity: s.Capacity,
				Message:  fmt.Sprintf("Slot %s: %d/%d osób", label, assigned, s.Capacity),
			})
		}

		if assigned > s.Capacity {
			result.Issues = append(result.Issues, ValidationIssue{
				Type:     "slot_overstaffed",
				Severity: "warning",
				Slot:     label,
				SlotID:   &s.ID,
				Assigned: assigned,
				Capacity: s.Capacity,
				Message:  fmt.Sprintf("Slot %s przeobsadzony: %d/%d osób", label, assigned, s.Capacity),
			})
		}
	}

	// Set Valid based on error-severity issues
	for _, issue := range result.Issues {
		if issue.Severity == "error" {
			result.Valid = false
			break
		}
	}

	return result
}

func checkDoubleBooked(slots []Slot, v Volunteer, result *ValidationResult) {
	for i := 0; i < len(slots); i++ {
		for j := i + 1; j < len(slots); j++ {
			a, b := slots[i], slots[j]
			// Overlap: a starts before b ends AND b starts before a ends (excluding touching)
			if a.StartTime.Before(b.EndTime) && b.StartTime.Before(a.EndTime) &&
				!a.EndTime.Equal(b.StartTime) && !b.EndTime.Equal(a.StartTime) {
				result.Valid = false
				labelA := slotLabel(a)
				labelB := slotLabel(b)
				result.Issues = append(result.Issues, ValidationIssue{
					Type:        "double_booked",
					Severity:    "error",
					Volunteer:   v.Nickname,
					VolunteerID: &v.ID,
					Slot:        fmt.Sprintf("%s / %s", labelA, labelB),
					Message:     fmt.Sprintf("%s przypisany do nakładających się slotów: %s i %s", v.Nickname, labelA, labelB),
				})
			}
		}
	}
}

func checkContinuousAndBreaks(slots []Slot, v Volunteer, result *ValidationResult) {
	if len(slots) < 2 {
		return
	}

	sorted := make([]Slot, len(slots))
	copy(sorted, slots)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartTime.Before(sorted[j].StartTime)
	})

	// Check consecutive hours (adjacent slots forming chains > 6h)
	chainStart := 0
	for i := 1; i < len(sorted); i++ {
		if sorted[i].StartTime.Equal(sorted[i-1].EndTime) {
			// Adjacent — continue chain
			var chainHours float64
			for j := chainStart; j <= i; j++ {
				chainHours += sorted[j].CreditHours
			}
			if chainHours > maxContinuousHours {
				label := slotLabel(sorted[chainStart])
				result.Issues = append(result.Issues, ValidationIssue{
					Type:        "consecutive_over_6h",
					Severity:    "warning",
					Volunteer:   v.Nickname,
					VolunteerID: &v.ID,
					Slot:        fmt.Sprintf("%s+%d slots", label, i-chainStart+1),
					Message:     fmt.Sprintf("%s: %.0fh ciągiem od %s", v.Nickname, chainHours, label),
				})
				chainStart = i
			}
		} else {
			chainStart = i

			// Not adjacent — check break duration
			gap := sorted[i].StartTime.Sub(sorted[i-1].EndTime)
			if gap > 0 && gap < time.Duration(minBreakHours)*time.Hour {
				result.Issues = append(result.Issues, ValidationIssue{
					Type:        "insufficient_break",
					Severity:    "warning",
					Volunteer:   v.Nickname,
					VolunteerID: &v.ID,
					Message:     fmt.Sprintf("%s: tylko %.0fh przerwy", v.Nickname, gap.Hours()),
				})
			}
		}
	}
}

func slotLabel(s Slot) string {
	if s.Label != nil {
		return *s.Label
	}
	return fmt.Sprintf("%s %s-%s", s.SlotType,
		s.StartTime.Format("15:04"),
		s.EndTime.Format("15:04"))
}
