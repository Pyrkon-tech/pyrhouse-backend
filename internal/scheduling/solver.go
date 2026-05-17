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


// volState tracks assignment state per volunteer during solving.
type volState struct {
	assignedHours float64
	assignedIDs   map[int]bool      // O(1) membership check
	slotByEnd     map[time.Time]int // endTime → slotID, for adjacentHoursBefore
	slotByStart   map[time.Time]int // startTime → slotID, for adjacentHoursAfter
}

func newVolState() *volState {
	return &volState{
		assignedIDs: make(map[int]bool),
		slotByEnd:   make(map[time.Time]int),
		slotByStart: make(map[time.Time]int),
	}
}

func (vs *volState) assign(s Slot) {
	vs.assignedHours += s.CreditHours
	vs.assignedIDs[s.ID] = true
	vs.slotByEnd[s.EndTime] = s.ID
	vs.slotByStart[s.StartTime] = s.ID
}

// Solve assigns volunteers to slots.
//
// Goals:
//   - Fair distribution: always pick the volunteer with the least assigned hours
//   - Block preference: try 4h blocks first, then 2h, then 1h as last resort
//   - Constraints: max 6h continuous, min 8h break between non-adjacent shifts
//
// All slot types (festival, montage, demontage) are treated uniformly:
// 1h blocks allow the same block-filling algorithm to work for every slot type.
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
		state[v.ID] = newVolState()
	}
	slotFill := make(map[int]int) // slot.ID → assigned count

	// Sort all slots by start time — block-fill works on the unified list.
	allSlots := make([]Slot, len(slots))
	copy(allSlots, slots)
	sort.Slice(allSlots, func(i, j int) bool {
		return allSlots[i].StartTime.Before(allSlots[j].StartTime)
	})

	var assignments []Assignment

	// preferred block sizes in descending order
	blockSizes := []int{4, 2, 1}

	// Iterate until no slot can be filled further.
	// We use a classic index loop (not range) so we can skip ahead after
	// assigning a block — this avoids staggered-overlap where two consecutive
	// block-starts (A:0-3, B:1-4) fill the middle slots to capacity while
	// leaving slot 0 with fill=1 but unable to host a full block.
	progress := true
	for progress {
		progress = false
		for i := 0; i < len(allSlots); i++ {
			slot := allSlots[i]
			if slotFill[slot.ID] >= slot.Capacity {
				continue
			}

			v := pickMinHoursVolunteer(volunteers, state, slotByID, allSlots, i, slotFill)
			if v == nil {
				continue
			}

			blockLen := 0
			for _, size := range blockSizes {
				if canAssignBlock(v, state[v.ID], slotByID, allSlots, i, size, slotFill) {
					blockLen = size
					break
				}
			}
			if blockLen == 0 {
				continue
			}

			for b := 0; b < blockLen; b++ {
				s := allSlots[i+b]
				assignments = append(assignments, Assignment{
					SlotID:      s.ID,
					VolunteerID: v.ID,
				})
				state[v.ID].assign(s)
				slotFill[s.ID]++
			}
			// Skip slots already covered by this block so the next iteration
			// starts a fresh non-overlapping block.
			i += blockLen - 1
			progress = true
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
// chain ending exactly at t (walking backwards). O(chain length) via slotByEnd index.
func adjacentHoursBefore(vs *volState, slotByID map[int]Slot, t time.Time) float64 {
	total := 0.0
	cur := t
	for {
		sid, ok := vs.slotByEnd[cur]
		if !ok {
			break
		}
		s := slotByID[sid]
		total += s.CreditHours
		cur = s.StartTime
	}
	return total
}

// adjacentHoursAfter sums credit hours of assigned slots that form a continuous
// chain starting exactly at t (walking forwards). O(chain length) via slotByStart index.
func adjacentHoursAfter(vs *volState, slotByID map[int]Slot, t time.Time) float64 {
	total := 0.0
	cur := t
	for {
		sid, ok := vs.slotByStart[cur]
		if !ok {
			break
		}
		s := slotByID[sid]
		total += s.CreditHours
		cur = s.EndTime
	}
	return total
}

// respectsBreak checks that the min 8h break constraint holds between `slot`
// and all non-adjacent assigned slots.
func respectsBreak(vs *volState, slotByID map[int]Slot, slot Slot) bool {
	for sid := range vs.assignedIDs {
		assigned, ok := slotByID[sid]
		if !ok {
			continue
		}
		if assigned.EndTime.Equal(slot.StartTime) || slot.EndTime.Equal(assigned.StartTime) {
			continue
		}
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
	return vs.assignedIDs[slotID]
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
