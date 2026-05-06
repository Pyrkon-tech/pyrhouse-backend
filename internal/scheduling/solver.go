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

	festivalHours := festivalEnd.Sub(festivalStart).Hours()
	numFestivalSlots := int(math.Ceil(festivalHours / float64(defaultShiftHours)))
	dayCapacity, nightCapacity := calcFestivalCapacity(volunteerCount, numFestivalSlots)

	current := festivalStart
	for current.Before(festivalEnd) {
		end := current.Add(time.Duration(defaultShiftHours) * time.Hour)
		if end.After(festivalEnd) {
			end = festivalEnd
		}
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

	// Demontage: last day of event
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

func calcFestivalCapacity(volunteerCount, numSlots int) (dayCapacity, nightCapacity int) {
	if numSlots == 0 {
		return 2, 2
	}
	festivalVolHours := float64(volunteerCount) * 14 * 0.6
	avgCapacity := festivalVolHours / (float64(numSlots) * float64(defaultShiftHours))

	dayCapacity = int(math.Ceil(avgCapacity))
	if dayCapacity < 2 {
		dayCapacity = 2
	}
	nightCapacity = int(math.Ceil(float64(dayCapacity) * 0.5))
	if nightCapacity < 2 {
		nightCapacity = 2
	}
	return dayCapacity, nightCapacity
}

// volState tracks assignment state per volunteer during solving.
type volState struct {
	assignedHours float64
	assignedIDs   []int // slot IDs assigned so far
}

// Solve assigns volunteers to slots.
//
// Goals:
//   - Fair distribution: always pick the volunteer with the least assigned hours
//   - Block preference: try 4h blocks first, then 2h, then 1h as last resort
//   - Constraints: max 6h continuous, min 8h break between non-adjacent shifts
func Solve(slots []Slot, volunteers []Volunteer) []Assignment {
	if len(slots) == 0 || len(volunteers) == 0 {
		return nil
	}

	// Index slots by ID for fast lookup
	slotByID := make(map[int]Slot, len(slots))
	for _, s := range slots {
		slotByID[s.ID] = s
	}

	state := make(map[int]*volState, len(volunteers))
	for _, v := range volunteers {
		state[v.ID] = &volState{}
	}
	slotFill := make(map[int]int) // slot.ID → assigned count

	// Separate festival slots and sort by start time
	var festSlots []Slot
	var otherSlots []Slot
	for _, s := range slots {
		if s.SlotType == SlotTypeFestival {
			festSlots = append(festSlots, s)
		} else {
			otherSlots = append(otherSlots, s)
		}
	}
	sort.Slice(festSlots, func(i, j int) bool {
		return festSlots[i].StartTime.Before(festSlots[j].StartTime)
	})

	var assignments []Assignment

	// preferred block sizes in descending order
	blockSizes := []int{4, 2, 1}

	// Iterate until no slot can be filled further
	progress := true
	for progress {
		progress = false
		for i := range festSlots {
			slot := festSlots[i]
			if slotFill[slot.ID] >= slot.Capacity {
				continue
			}

			v := pickMinHoursVolunteer(volunteers, state, slotByID, festSlots, i, slotFill)
			if v == nil {
				continue
			}

			blockLen := 0
			for _, size := range blockSizes {
				if canAssignBlock(v, state[v.ID], slotByID, festSlots, i, size, slotFill) {
					blockLen = size
					break
				}
			}
			if blockLen == 0 {
				continue
			}

			for b := 0; b < blockLen; b++ {
				s := festSlots[i+b]
				assignments = append(assignments, Assignment{
					SlotID:      s.ID,
					VolunteerID: v.ID,
				})
				state[v.ID].assignedHours += s.CreditHours
				state[v.ID].assignedIDs = append(state[v.ID].assignedIDs, s.ID)
				slotFill[s.ID]++
			}
			progress = true
		}
	}

	// Montage / demontage: greedy, existing logic
	for _, slot := range otherSlots {
		for i := range volunteers {
			v := volunteers[i]
			if slotFill[slot.ID] >= slot.Capacity {
				break
			}
			vs := state[v.ID]
			if vs.assignedHours >= float64(v.TargetHours)+2 {
				continue
			}
			if v.AvailableFrom.After(slot.EndTime) || v.AvailableTo.Before(slot.StartTime) {
				continue
			}
			alreadyIn := false
			for _, sid := range vs.assignedIDs {
				if sid == slot.ID {
					alreadyIn = true
					break
				}
			}
			if alreadyIn {
				continue
			}
			assignments = append(assignments, Assignment{
				SlotID:      slot.ID,
				VolunteerID: v.ID,
			})
			vs.assignedHours += slot.CreditHours
			vs.assignedIDs = append(vs.assignedIDs, slot.ID)
			slotFill[slot.ID]++
		}
	}

	return assignments
}

// pickMinHoursVolunteer returns the eligible volunteer with the fewest assigned hours
// for the slot at festSlots[startIdx].
func pickMinHoursVolunteer(
	volunteers []Volunteer,
	state map[int]*volState,
	slotByID map[int]Slot,
	festSlots []Slot,
	startIdx int,
	slotFill map[int]int,
) *Volunteer {
	slot := festSlots[startIdx]
	var best *Volunteer
	bestHours := math.MaxFloat64

	for i := range volunteers {
		v := &volunteers[i]
		vs := state[v.ID]

		if vs.assignedHours >= float64(v.TargetHours)+2 {
			continue
		}
		if slot.StartTime.Before(v.AvailableFrom) || slot.EndTime.After(v.AvailableTo) {
			continue
		}
		if isAlreadyIn(vs, slot.ID) {
			continue
		}
		if !respectsBreak(vs, slotByID, slot) {
			continue
		}

		if vs.assignedHours < bestHours {
			bestHours = vs.assignedHours
			best = v
		}
	}
	return best
}

// canAssignBlock checks whether volunteer v can be assigned a block of blockLen
// consecutive festival slots starting at festSlots[startIdx].
func canAssignBlock(
	v *Volunteer,
	vs *volState,
	slotByID map[int]Slot,
	festSlots []Slot,
	startIdx, blockLen int,
	slotFill map[int]int,
) bool {
	if startIdx+blockLen > len(festSlots) {
		return false
	}

	// Check that slots in the block are consecutive (no gaps)
	for b := 1; b < blockLen; b++ {
		prev := festSlots[startIdx+b-1]
		cur := festSlots[startIdx+b]
		if !prev.EndTime.Equal(cur.StartTime) {
			return false
		}
	}

	// All slots in block must still need volunteers and volunteer must not be in them
	var blockHours float64
	for b := 0; b < blockLen; b++ {
		s := festSlots[startIdx+b]
		if slotFill[s.ID] >= s.Capacity {
			return false
		}
		if isAlreadyIn(vs, s.ID) {
			return false
		}
		if s.StartTime.Before(v.AvailableFrom) || s.EndTime.After(v.AvailableTo) {
			return false
		}
		blockHours += s.CreditHours
	}

	// Would exceed target+2h?
	if vs.assignedHours+blockHours > float64(v.TargetHours)+2 {
		return false
	}

	// Continuous run constraint: count adjacent already-assigned hours + block
	blockStart := festSlots[startIdx].StartTime
	blockEnd := festSlots[startIdx+blockLen-1].EndTime
	continuousHours := blockHours

	// Extend left: find assigned slots immediately before blockStart
	continuousHours += adjacentHoursBefore(vs, slotByID, blockStart)

	// Extend right: find assigned slots immediately after blockEnd
	continuousHours += adjacentHoursAfter(vs, slotByID, blockEnd)

	if continuousHours > maxContinuousHours {
		return false
	}

	// Min break constraint for all slots in the block vs non-adjacent assigned slots
	for b := 0; b < blockLen; b++ {
		s := festSlots[startIdx+b]
		if !respectsBreak(vs, slotByID, s) {
			return false
		}
	}

	return true
}

// adjacentHoursBefore sums credit hours of assigned slots that form a continuous
// chain ending exactly at t (walking backwards).
func adjacentHoursBefore(vs *volState, slotByID map[int]Slot, t time.Time) float64 {
	total := 0.0
	cur := t
	for {
		found := false
		for _, sid := range vs.assignedIDs {
			s, ok := slotByID[sid]
			if ok && s.EndTime.Equal(cur) {
				total += s.CreditHours
				cur = s.StartTime
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return total
}

// adjacentHoursAfter sums credit hours of assigned slots that form a continuous
// chain starting exactly at t (walking forwards).
func adjacentHoursAfter(vs *volState, slotByID map[int]Slot, t time.Time) float64 {
	total := 0.0
	cur := t
	for {
		found := false
		for _, sid := range vs.assignedIDs {
			s, ok := slotByID[sid]
			if ok && s.StartTime.Equal(cur) {
				total += s.CreditHours
				cur = s.EndTime
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return total
}

// respectsBreak checks that the min 8h break constraint holds between `slot`
// and all non-adjacent assigned slots.
func respectsBreak(vs *volState, slotByID map[int]Slot, slot Slot) bool {
	for _, sid := range vs.assignedIDs {
		assigned, ok := slotByID[sid]
		if !ok {
			continue
		}
		// Adjacent slots — handled by continuous check, not a break violation
		if assigned.EndTime.Equal(slot.StartTime) || slot.EndTime.Equal(assigned.StartTime) {
			continue
		}
		// Overlapping — blocked
		if slot.StartTime.Before(assigned.EndTime) && assigned.StartTime.Before(slot.EndTime) {
			return false
		}

		var gap time.Duration
		if slot.StartTime.After(assigned.EndTime) {
			gap = slot.StartTime.Sub(assigned.EndTime)
		} else {
			gap = assigned.StartTime.Sub(slot.EndTime)
		}
		if gap < time.Duration(minBreakHours)*time.Hour {
			return false
		}
	}
	return true
}

func isAlreadyIn(vs *volState, slotID int) bool {
	for _, sid := range vs.assignedIDs {
		if sid == slotID {
			return true
		}
	}
	return false
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
