package scheduling

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	maxContinuousHours = 6.0
	minBreakHours      = 8.0
	defaultShiftHours  = 4
)

// GenerateSlots creates time slots from schedule dates.
// volunteerCount is used to calculate dynamic festival capacity.
func GenerateSlots(schedule *Schedule, volunteerCount int) []Slot {
	var slots []Slot

	// Montage days: event_start to day before festival (Tue, Wed, Thu)
	montageStart := schedule.EventStart
	// Compare dates only (strip time) to avoid midnight vs 10:00 issues
	festivalDate := time.Date(schedule.FestivalStart.Year(), schedule.FestivalStart.Month(), schedule.FestivalStart.Day(), 0, 0, 0, 0, schedule.FestivalStart.Location())

	for d := montageStart; d.Before(festivalDate); d = d.AddDate(0, 0, 1) {
		dayStart := time.Date(d.Year(), d.Month(), d.Day(), 8, 0, 0, 0, d.Location())
		dayEnd := time.Date(d.Year(), d.Month(), d.Day(), 20, 0, 0, 0, d.Location())
		label := fmt.Sprintf("Montaż - %s", polishWeekday(d.Weekday()))
		slots = append(slots, Slot{
			ScheduleID:  schedule.ID,
			SlotType:    SlotTypeMontage,
			StartTime:   dayStart,
			EndTime:     dayEnd,
			CreditHours: 7,
			Capacity:    8,
			Label:       &label,
		})
	}

	// Festival slots: continuous 4h blocks from festival_start to festival_end (24/7 operation)
	festivalStart := schedule.FestivalStart
	festivalEnd := schedule.FestivalEnd

	// Calculate dynamic capacity based on volunteer count
	festivalHours := festivalEnd.Sub(festivalStart).Hours()
	numFestivalSlots := int(math.Ceil(festivalHours / float64(defaultShiftHours)))
	dayCapacity, nightCapacity := calcFestivalCapacity(volunteerCount, numFestivalSlots)

	current := festivalStart
	for current.Before(festivalEnd) {
		end := current.Add(time.Duration(defaultShiftHours) * time.Hour)
		if end.After(festivalEnd) {
			end = festivalEnd
		}
		// Skip tiny leftover slots (< 1h)
		if end.Sub(current) < time.Hour {
			break
		}

		hour := current.Hour()
		capacity := dayCapacity
		if hour >= 22 || hour < 6 {
			capacity = nightCapacity
		}

		label := formatSlotLabel(current, end)
		creditHours := end.Sub(current).Hours()

		slots = append(slots, Slot{
			ScheduleID:  schedule.ID,
			SlotType:    SlotTypeFestival,
			StartTime:   current,
			EndTime:     end,
			CreditHours: creditHours,
			Capacity:    capacity,
			Label:       &label,
		})

		current = end
	}

	// Demontage: Monday (day after festival_end or event_end)
	eventEnd := schedule.EventEnd
	demonLabel := fmt.Sprintf("Demontaż - %s", polishWeekday(eventEnd.Weekday()))
	dayStart := time.Date(eventEnd.Year(), eventEnd.Month(), eventEnd.Day(), 8, 0, 0, 0, eventEnd.Location())
	dayEnd := time.Date(eventEnd.Year(), eventEnd.Month(), eventEnd.Day(), 20, 0, 0, 0, eventEnd.Location())
	slots = append(slots, Slot{
		ScheduleID:  schedule.ID,
		SlotType:    SlotTypeDemontage,
		StartTime:   dayStart,
		EndTime:     dayEnd,
		CreditHours: 7,
		Capacity:    3,
		Label:       &demonLabel,
	})

	return slots
}

// calcFestivalCapacity calculates day and night capacity for festival slots
// based on volunteer count and number of slots.
func calcFestivalCapacity(volunteerCount, numSlots int) (dayCapacity, nightCapacity int) {
	if numSlots == 0 {
		return 2, 2
	}
	// Avg capacity = volunteers * avgTargetHours / totalFestivalSlotHours
	// Assume ~60% of target hours go to festival (rest to montage/demontage)
	festivalVolHours := float64(volunteerCount) * 14 * 0.6
	avgCapacity := festivalVolHours / (float64(numSlots) * float64(defaultShiftHours))

	dayCapacity = int(math.Ceil(avgCapacity))
	if dayCapacity < 2 {
		dayCapacity = 2
	}

	// Night capacity: half of day, minimum 2
	nightCapacity = int(math.Ceil(float64(dayCapacity) * 0.5))
	if nightCapacity < 2 {
		nightCapacity = 2
	}

	return dayCapacity, nightCapacity
}

type volState struct {
	assignedHours float64
	assignedSlots []int // indices into slots
}

// Solve assigns volunteers to slots using a greedy algorithm.
func Solve(slots []Slot, volunteers []Volunteer) []Assignment {
	// Sort volunteers by availability window (most constrained first)
	sorted := make([]Volunteer, len(volunteers))
	copy(sorted, volunteers)
	sort.Slice(sorted, func(i, j int) bool {
		windowI := sorted[i].AvailableTo.Sub(sorted[i].AvailableFrom)
		windowJ := sorted[j].AvailableTo.Sub(sorted[j].AvailableFrom)
		return windowI < windowJ
	})

	state := make(map[int]*volState) // volunteer ID → state
	for _, v := range sorted {
		state[v.ID] = &volState{}
	}

	slotFill := make(map[int]int) // slot index → how many assigned

	// Sort slots: festival first, then montage/demontage (prefer festival assignments)
	slotOrder := make([]int, len(slots))
	for i := range slots {
		slotOrder[i] = i
	}
	sort.Slice(slotOrder, func(i, j int) bool {
		si, sj := slots[slotOrder[i]], slots[slotOrder[j]]
		// Festival slots first
		if si.SlotType == SlotTypeFestival && sj.SlotType != SlotTypeFestival {
			return true
		}
		if si.SlotType != SlotTypeFestival && sj.SlotType == SlotTypeFestival {
			return false
		}
		return si.StartTime.Before(sj.StartTime)
	})

	var assignments []Assignment

	// Pass 1: Assign festival slots (everyone should have some)
	for _, si := range slotOrder {
		slot := slots[si]
		if slot.SlotType != SlotTypeFestival {
			continue
		}
		assignments = assignToSlot(slot, si, sorted, state, slotFill, slots, assignments)
	}

	// Pass 2: Assign montage/demontage to fill remaining hours
	for _, si := range slotOrder {
		slot := slots[si]
		if slot.SlotType == SlotTypeFestival {
			continue
		}
		assignments = assignToSlot(slot, si, sorted, state, slotFill, slots, assignments)
	}

	return assignments
}

func assignToSlot(slot Slot, slotIdx int, volunteers []Volunteer, state map[int]*volState, slotFill map[int]int, allSlots []Slot, assignments []Assignment) []Assignment {
	for _, v := range volunteers {
		if slotFill[slotIdx] >= slot.Capacity {
			break
		}

		vs := state[v.ID]
		if vs.assignedHours >= float64(v.TargetHours) {
			continue
		}

		if !canAssign(v, slot, slotIdx, vs, allSlots) {
			continue
		}

		assignments = append(assignments, Assignment{
			SlotID:      slot.ID,
			VolunteerID: v.ID,
		})
		vs.assignedHours += slot.CreditHours
		vs.assignedSlots = append(vs.assignedSlots, slotIdx)
		slotFill[slotIdx]++
	}
	return assignments
}

func canAssign(v Volunteer, slot Slot, slotIdx int, vs *volState, allSlots []Slot) bool {
	// Check availability window
	if slot.SlotType == SlotTypeMontage || slot.SlotType == SlotTypeDemontage {
		// Montage/demontage: "show up when you can" — overlap is enough
		if v.AvailableFrom.After(slot.EndTime) || v.AvailableTo.Before(slot.StartTime) {
			return false
		}
	} else {
		// Festival: strict containment — volunteer must cover the entire shift
		if slot.StartTime.Before(v.AvailableFrom) || slot.EndTime.After(v.AvailableTo) {
			return false
		}
	}

	// Check not already in this slot (compare by index, not ID, since ID=0 for generated slots)
	for _, si := range vs.assignedSlots {
		if si == slotIdx {
			return false
		}
	}

	// Check max continuous hours (no more than 6h in a row)
	// Skip for montage/demontage — they are all-day shifts with informal breaks
	if slot.SlotType == SlotTypeFestival {
		continuousHours := slot.CreditHours
		for _, si := range vs.assignedSlots {
			assigned := allSlots[si]
			if assigned.EndTime.Equal(slot.StartTime) || slot.EndTime.Equal(assigned.StartTime) {
				continuousHours += assigned.CreditHours
			}
		}
		if continuousHours > maxContinuousHours {
			return false
		}
	}

	// Check min break (8h between non-adjacent shifts)
	for _, si := range vs.assignedSlots {
		assigned := allSlots[si]
		// Skip adjacent (already handled by continuous check)
		if assigned.EndTime.Equal(slot.StartTime) || slot.EndTime.Equal(assigned.StartTime) {
			continue
		}

		var gap time.Duration
		if slot.StartTime.After(assigned.EndTime) {
			gap = slot.StartTime.Sub(assigned.EndTime)
		} else if assigned.StartTime.After(slot.EndTime) {
			gap = assigned.StartTime.Sub(slot.EndTime)
		} else {
			// Overlapping — can't assign
			return false
		}

		if gap > 0 && gap < time.Duration(minBreakHours)*time.Hour {
			return false
		}
	}

	// Would exceed target hours?
	if vs.assignedHours+slot.CreditHours > float64(v.TargetHours)+2 {
		// Allow small overshoot (up to 2h) but not more
		return false
	}

	return true
}

func polishWeekday(w time.Weekday) string {
	days := map[time.Weekday]string{
		time.Monday:    "Poniedziałek",
		time.Tuesday:   "Wtorek",
		time.Wednesday: "Środa",
		time.Thursday:  "Czwartek",
		time.Friday:    "Piątek",
		time.Saturday:  "Sobota",
		time.Sunday:    "Niedziela",
	}
	return days[w]
}

func formatSlotLabel(start, end time.Time) string {
	dayAbbr := map[time.Weekday]string{
		time.Monday: "Pn", time.Tuesday: "Wt", time.Wednesday: "Śr",
		time.Thursday: "Czw", time.Friday: "Pt", time.Saturday: "Sb", time.Sunday: "Nd",
	}
	return fmt.Sprintf("%s %02d:%02d-%02d:%02d",
		dayAbbr[start.Weekday()],
		start.Hour(), start.Minute(),
		end.Hour(), end.Minute())
}
