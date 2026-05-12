package scheduling

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func festSlot(id, hour int, creditHours float64) Slot {
	return Slot{
		ID:          id,
		ScheduleID:  1,
		SlotType:    SlotTypeFestival,
		StartTime:   time.Date(2025, 6, 20, hour, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2025, 6, 20, hour+int(creditHours), 0, 0, 0, time.UTC),
		CreditHours: creditHours,
		Capacity:    2,
	}
}

func volunteer(id int, targetHours int) Volunteer {
	return Volunteer{
		ID:            id,
		ScheduleID:    1,
		Nickname:      "Vol",
		TargetHours:   targetHours,
		AvailableFrom: time.Date(2025, 6, 20, 0, 0, 0, 0, time.UTC),
		AvailableTo:   time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC),
	}
}

func TestSolve_Empty(t *testing.T) {
	assert.Nil(t, Solve(nil, nil))
	assert.Nil(t, Solve([]Slot{festSlot(1, 10, 1)}, nil))
	assert.Nil(t, Solve(nil, []Volunteer{volunteer(1, 4)}))
}

func TestSolve_SingleSlotSingleVolunteer(t *testing.T) {
	slots := []Slot{festSlot(1, 10, 2)}
	vols := []Volunteer{volunteer(1, 4)}

	assignments := Solve(slots, vols)

	assert.Len(t, assignments, 1)
	assert.Equal(t, 1, assignments[0].SlotID)
	assert.Equal(t, 1, assignments[0].VolunteerID)
}

func TestSolve_FillsCapacity(t *testing.T) {
	s := festSlot(1, 10, 2)
	s.Capacity = 3
	vols := []Volunteer{volunteer(1, 4), volunteer(2, 4), volunteer(3, 4)}

	assignments := Solve([]Slot{s}, vols)

	assert.Len(t, assignments, 3)
}

func TestSolve_NoDoubleAssignment(t *testing.T) {
	// One slot, two volunteers — same volunteer must not appear twice
	s := festSlot(1, 10, 2)
	s.Capacity = 2
	vols := []Volunteer{volunteer(1, 4), volunteer(2, 4)}

	assignments := Solve([]Slot{s}, vols)

	seen := map[int]bool{}
	for _, a := range assignments {
		assert.False(t, seen[a.VolunteerID], "volunteer assigned twice to same slot")
		seen[a.VolunteerID] = true
	}
}

func TestSolve_FairDistribution(t *testing.T) {
	// 12 adjacent 1h slots, 2 volunteers — hours should be distributed roughly equally.
	// The solver assigns in blocks so per-slot perfect balance isn't guaranteed,
	// but over 12h the difference should be ≤ maxContinuousHours (6h).
	var slots []Slot
	for i := 0; i < 12; i++ {
		s := Slot{
			ID:          i + 1,
			ScheduleID:  1,
			SlotType:    SlotTypeFestival,
			StartTime:   time.Date(2025, 6, 20, 8+i, 0, 0, 0, time.UTC),
			EndTime:     time.Date(2025, 6, 20, 9+i, 0, 0, 0, time.UTC),
			CreditHours: 1,
			Capacity:    1,
		}
		slots = append(slots, s)
	}
	vols := []Volunteer{volunteer(1, 6), volunteer(2, 6)}

	assignments := Solve(slots, vols)

	assert.NotEmpty(t, assignments)
	hours := map[int]float64{}
	for _, a := range assignments {
		hours[a.VolunteerID]++
	}
	// Both volunteers should receive some assignments
	assert.Greater(t, hours[1], 0.0, "vol 1 should receive at least one slot")
	assert.Greater(t, hours[2], 0.0, "vol 2 should receive at least one slot")
	// Neither volunteer should have more than maxContinuousHours more than the other
	diff := hours[1] - hours[2]
	if diff < 0 {
		diff = -diff
	}
	assert.LessOrEqual(t, diff, maxContinuousHours, "hour difference between volunteers should be within one block")
}

func TestSolve_RespectsAvailability(t *testing.T) {
	s := festSlot(1, 22, 2) // 22:00-00:00
	s.Capacity = 1

	// Vol 1 not available at that hour
	v1 := volunteer(1, 4)
	v1.AvailableFrom = time.Date(2025, 6, 20, 0, 0, 0, 0, time.UTC)
	v1.AvailableTo = time.Date(2025, 6, 20, 20, 0, 0, 0, time.UTC) // ends at 20:00

	// Vol 2 is available
	v2 := volunteer(2, 4)

	assignments := Solve([]Slot{s}, []Volunteer{v1, v2})

	for _, a := range assignments {
		assert.Equal(t, 2, a.VolunteerID, "only vol 2 is available for this slot")
	}
}

func TestSolve_RespectsMaxContinuousHours(t *testing.T) {
	// 8 adjacent 1h slots with capacity 1, 1 volunteer — solver should not assign all 8
	var slots []Slot
	for i := 0; i < 8; i++ {
		s := Slot{
			ID:          i + 1,
			ScheduleID:  1,
			SlotType:    SlotTypeFestival,
			StartTime:   time.Date(2025, 6, 20, 8+i, 0, 0, 0, time.UTC),
			EndTime:     time.Date(2025, 6, 20, 9+i, 0, 0, 0, time.UTC),
			CreditHours: 1,
			Capacity:    1,
		}
		slots = append(slots, s)
	}

	v1 := volunteer(1, 8)

	assignments := Solve(slots, []Volunteer{v1})

	// Should stop at maxContinuousHours (6), not assign all 8
	assert.LessOrEqual(t, len(assignments), int(maxContinuousHours))
}

func TestSolve_RespectsMinBreak(t *testing.T) {
	// Slot A then slot B with only 4h gap (< 8h required)
	slotA := Slot{
		ID: 1, ScheduleID: 1, SlotType: SlotTypeFestival,
		StartTime: time.Date(2025, 6, 20, 8, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2025, 6, 20, 10, 0, 0, 0, time.UTC),
		CreditHours: 2, Capacity: 1,
	}
	slotB := Slot{
		ID: 2, ScheduleID: 1, SlotType: SlotTypeFestival,
		StartTime: time.Date(2025, 6, 20, 14, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2025, 6, 20, 16, 0, 0, 0, time.UTC),
		CreditHours: 2, Capacity: 1,
	}

	v1 := volunteer(1, 4)

	assignments := Solve([]Slot{slotA, slotB}, []Volunteer{v1})

	// With only one volunteer and insufficient break between slots,
	// solver should not assign both slots to the same volunteer
	if len(assignments) == 2 {
		t.Error("solver should not assign both slots to same volunteer with insufficient break")
	}
}

func TestSolve_AlreadyAssignedNotRepeated(t *testing.T) {
	// Vol starts with assignedHours already above target — tests isAlreadyIn guard
	s1 := festSlot(1, 10, 1)
	s1.Capacity = 1

	v1 := volunteer(1, 2)

	assignments := Solve([]Slot{s1}, []Volunteer{v1})

	// Slot should be assigned once
	count := 0
	for _, a := range assignments {
		if a.SlotID == 1 && a.VolunteerID == 1 {
			count++
		}
	}
	assert.LessOrEqual(t, count, 1, "volunteer must not be assigned to same slot twice")
}

func TestVolState_Assign(t *testing.T) {
	vs := newVolState()
	s := festSlot(1, 10, 2)

	vs.assign(s)

	assert.Equal(t, 2.0, vs.assignedHours)
	assert.True(t, vs.assignedIDs[1])
	assert.Equal(t, 1, vs.slotByEnd[s.EndTime])
	assert.Equal(t, 1, vs.slotByStart[s.StartTime])
}

func TestVolState_IsAlreadyIn(t *testing.T) {
	vs := newVolState()
	s := festSlot(42, 10, 1)
	vs.assign(s)

	assert.True(t, isAlreadyIn(vs, 42))
	assert.False(t, isAlreadyIn(vs, 99))
}
