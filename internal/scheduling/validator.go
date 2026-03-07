package scheduling

import (
	"fmt"
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
	volAssignments := make(map[int][]int) // volunteer ID → slot IDs
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
				Type:      "under_hours",
				Volunteer: v.Nickname,
				Assigned:  int(totalHours),
				Target:    v.TargetHours,
			})
		}

		// No festival shifts
		if !hasFestival && len(slotIDs) > 0 {
			result.Issues = append(result.Issues, ValidationIssue{
				Type:      "no_festival_shifts",
				Volunteer: v.Nickname,
			})
		}

		// Check consecutive hours and break constraints
		checkContinuousAndBreaks(assignedSlots, v, result)
	}

	// Check slot staffing
	for _, s := range slots {
		assigned := len(slotAssignments[s.ID])
		if assigned < s.Capacity {
			label := ""
			if s.Label != nil {
				label = *s.Label
			}
			result.Issues = append(result.Issues, ValidationIssue{
				Type:     "slot_understaffed",
				Slot:     label,
				Assigned: assigned,
				Capacity: s.Capacity,
			})
		}
	}

	return result
}

func checkContinuousAndBreaks(slots []Slot, v Volunteer, result *ValidationResult) {
	if len(slots) < 2 {
		return
	}

	// Sort by start time
	sorted := make([]Slot, len(slots))
	copy(sorted, slots)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].StartTime.Before(sorted[i].StartTime) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

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
				result.Valid = false
				label := ""
				if sorted[chainStart].Label != nil {
					label = *sorted[chainStart].Label
				}
				result.Issues = append(result.Issues, ValidationIssue{
					Type:      "consecutive_over_6h",
					Volunteer: v.Nickname,
					Slot:      fmt.Sprintf("%s+%d slots", label, i-chainStart+1),
				})
				chainStart = i
			}
		} else {
			chainStart = i

			// Not adjacent — check break duration
			gap := sorted[i].StartTime.Sub(sorted[i-1].EndTime)
			if gap > 0 && gap < time.Duration(minBreakHours)*time.Hour {
				result.Issues = append(result.Issues, ValidationIssue{
					Type:      "insufficient_break",
					Volunteer: v.Nickname,
					Slot: fmt.Sprintf("%.0fh break between slots",
						gap.Hours()),
				})
			}
		}
	}
}
