package scheduling

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var utc = time.UTC

func ts(hour, min int) time.Time {
	return time.Date(2025, 6, 20, hour, min, 0, 0, utc)
}

func slot(id int, start, end time.Time, slotType string, capacity int, creditHours float64) Slot {
	return Slot{
		ID:          id,
		ScheduleID:  1,
		SlotType:    slotType,
		StartTime:   start,
		EndTime:     end,
		CreditHours: creditHours,
		Capacity:    capacity,
	}
}

func vol(id int, from, to time.Time, targetHours int) Volunteer {
	return Volunteer{
		ID:            id,
		ScheduleID:    1,
		Nickname:      "Vol" + string(rune('A'+id-1)),
		TargetHours:   targetHours,
		AvailableFrom: from,
		AvailableTo:   to,
	}
}

// issueTypes extracts issue Type fields for easy assertion.
func issueTypes(issues []ValidationIssue) []string {
	types := make([]string, len(issues))
	for i, iss := range issues {
		types[i] = iss.Type
	}
	return types
}

func TestValidate_NoIssues(t *testing.T) {
	s1 := slot(1, ts(10, 0), ts(12, 0), SlotTypeFestival, 1, 2)
	v1 := vol(1, ts(8, 0), ts(20, 0), 2)
	a1 := Assignment{SlotID: 1, VolunteerID: 1}

	result := Validate([]Slot{s1}, []Volunteer{v1}, []Assignment{a1})

	assert.True(t, result.Valid)
	assert.Empty(t, result.Issues)
}

func TestValidate_UnderHours(t *testing.T) {
	s1 := slot(1, ts(10, 0), ts(12, 0), SlotTypeFestival, 1, 2)
	v1 := vol(1, ts(8, 0), ts(20, 0), 8) // needs 8h, gets 2h
	a1 := Assignment{SlotID: 1, VolunteerID: 1}

	result := Validate([]Slot{s1}, []Volunteer{v1}, []Assignment{a1})

	assert.Contains(t, issueTypes(result.Issues), "under_hours")
}

func TestValidate_OverHours(t *testing.T) {
	// Three montage slots (7h credit each) = 21h total > 18h threshold.
	// Use different days to avoid consecutive_over_6h interference.
	day1 := time.Date(2025, 6, 18, 8, 0, 0, 0, utc)
	day2 := time.Date(2025, 6, 19, 8, 0, 0, 0, utc)
	day3 := time.Date(2025, 6, 20, 8, 0, 0, 0, utc)
	end1 := time.Date(2025, 6, 18, 20, 0, 0, 0, utc)
	end2 := time.Date(2025, 6, 19, 20, 0, 0, 0, utc)
	end3 := time.Date(2025, 6, 20, 20, 0, 0, 0, utc)

	s1 := slot(1, day1, end1, SlotTypeMontage, 1, 7)
	s2 := slot(2, day2, end2, SlotTypeMontage, 1, 7)
	s3 := slot(3, day3, end3, SlotTypeMontage, 1, 7)
	// Add one festival slot to satisfy no_festival_shifts check
	fest := slot(4, time.Date(2025, 6, 21, 10, 0, 0, 0, utc), time.Date(2025, 6, 21, 11, 0, 0, 0, utc), SlotTypeFestival, 1, 1)

	v1 := vol(1, time.Date(2025, 6, 18, 0, 0, 0, 0, utc), time.Date(2025, 6, 22, 0, 0, 0, 0, utc), 10)

	assignments := []Assignment{
		{SlotID: 1, VolunteerID: 1},
		{SlotID: 2, VolunteerID: 1},
		{SlotID: 3, VolunteerID: 1},
		{SlotID: 4, VolunteerID: 1},
	}

	result := Validate([]Slot{s1, s2, s3, fest}, []Volunteer{v1}, assignments)

	assert.Contains(t, issueTypes(result.Issues), "over_hours")
}

func TestValidate_NoFestivalShift(t *testing.T) {
	s1 := slot(1, ts(8, 0), ts(20, 0), SlotTypeMontage, 1, 7)
	v1 := vol(1, ts(6, 0), ts(22, 0), 7)
	a1 := Assignment{SlotID: 1, VolunteerID: 1}

	result := Validate([]Slot{s1}, []Volunteer{v1}, []Assignment{a1})

	assert.Contains(t, issueTypes(result.Issues), "no_festival_shifts")
}

func TestValidate_OutsideAvailability(t *testing.T) {
	s1 := slot(1, ts(6, 0), ts(8, 0), SlotTypeFestival, 1, 2) // starts before availability
	v1 := vol(1, ts(8, 0), ts(20, 0), 2)
	a1 := Assignment{SlotID: 1, VolunteerID: 1}

	result := Validate([]Slot{s1}, []Volunteer{v1}, []Assignment{a1})

	assert.Contains(t, issueTypes(result.Issues), "outside_availability")
}

func TestValidate_SlotUnderstaffed(t *testing.T) {
	s1 := slot(1, ts(10, 0), ts(12, 0), SlotTypeFestival, 2, 2) // capacity 2, nobody assigned
	v1 := vol(1, ts(8, 0), ts(20, 0), 0)

	result := Validate([]Slot{s1}, []Volunteer{v1}, []Assignment{})

	assert.Contains(t, issueTypes(result.Issues), "slot_understaffed")
}

func TestCheckDoubleBooked_Overlap(t *testing.T) {
	v1 := vol(1, ts(0, 0), ts(23, 0), 4)
	slots := []Slot{
		slot(1, ts(10, 0), ts(12, 0), SlotTypeFestival, 1, 2),
		slot(2, ts(11, 0), ts(13, 0), SlotTypeFestival, 1, 2), // overlaps slot 1
	}

	result := &ValidationResult{Valid: true}
	checkDoubleBooked(slots, v1, result)

	assert.False(t, result.Valid)
	assert.Len(t, result.Issues, 1)
	assert.Equal(t, "double_booked", result.Issues[0].Type)
	assert.Equal(t, "error", result.Issues[0].Severity)
}

func TestCheckDoubleBooked_Adjacent_NoOverlap(t *testing.T) {
	v1 := vol(1, ts(0, 0), ts(23, 0), 4)
	slots := []Slot{
		slot(1, ts(10, 0), ts(12, 0), SlotTypeFestival, 1, 2),
		slot(2, ts(12, 0), ts(14, 0), SlotTypeFestival, 1, 2), // touches but doesn't overlap
	}

	result := &ValidationResult{Valid: true}
	checkDoubleBooked(slots, v1, result)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Issues)
}

func TestCheckContinuousAndBreaks_Over6h(t *testing.T) {
	v1 := vol(1, ts(0, 0), ts(23, 0), 8)
	// 7 adjacent 1h festival slots = 7h continuous
	slots := []Slot{
		slot(1, ts(8, 0), ts(9, 0), SlotTypeFestival, 1, 1),
		slot(2, ts(9, 0), ts(10, 0), SlotTypeFestival, 1, 1),
		slot(3, ts(10, 0), ts(11, 0), SlotTypeFestival, 1, 1),
		slot(4, ts(11, 0), ts(12, 0), SlotTypeFestival, 1, 1),
		slot(5, ts(12, 0), ts(13, 0), SlotTypeFestival, 1, 1),
		slot(6, ts(13, 0), ts(14, 0), SlotTypeFestival, 1, 1),
		slot(7, ts(14, 0), ts(15, 0), SlotTypeFestival, 1, 1),
	}

	result := &ValidationResult{Valid: true}
	checkContinuousAndBreaks(slots, v1, result)

	assert.Contains(t, issueTypes(result.Issues), "consecutive_over_6h")
}

func TestCheckContinuousAndBreaks_Exactly6h_NoIssue(t *testing.T) {
	v1 := vol(1, ts(0, 0), ts(23, 0), 6)
	// exactly 6h continuous — should not trigger warning
	slots := []Slot{
		slot(1, ts(8, 0), ts(9, 0), SlotTypeFestival, 1, 1),
		slot(2, ts(9, 0), ts(10, 0), SlotTypeFestival, 1, 1),
		slot(3, ts(10, 0), ts(11, 0), SlotTypeFestival, 1, 1),
		slot(4, ts(11, 0), ts(12, 0), SlotTypeFestival, 1, 1),
		slot(5, ts(12, 0), ts(13, 0), SlotTypeFestival, 1, 1),
		slot(6, ts(13, 0), ts(14, 0), SlotTypeFestival, 1, 1),
	}

	result := &ValidationResult{Valid: true}
	checkContinuousAndBreaks(slots, v1, result)

	assert.NotContains(t, issueTypes(result.Issues), "consecutive_over_6h")
}

func TestCheckContinuousAndBreaks_InsufficientBreak(t *testing.T) {
	v1 := vol(1, ts(0, 0), ts(23, 0), 4)
	// 2h shift, then 4h gap (< 8h required), then another 2h shift
	slots := []Slot{
		slot(1, ts(8, 0), ts(10, 0), SlotTypeFestival, 1, 2),
		slot(2, ts(14, 0), ts(16, 0), SlotTypeFestival, 1, 2), // only 4h gap
	}

	result := &ValidationResult{Valid: true}
	checkContinuousAndBreaks(slots, v1, result)

	assert.Contains(t, issueTypes(result.Issues), "insufficient_break")
}

func TestCheckContinuousAndBreaks_SufficientBreak(t *testing.T) {
	v1 := vol(1, ts(0, 0), ts(23, 0), 4)
	// 2h shift, then 9h gap (> 8h required), then another 2h shift
	slots := []Slot{
		slot(1, ts(6, 0), ts(8, 0), SlotTypeFestival, 1, 2),
		slot(2, ts(17, 0), ts(19, 0), SlotTypeFestival, 1, 2), // 9h gap
	}

	result := &ValidationResult{Valid: true}
	checkContinuousAndBreaks(slots, v1, result)

	assert.NotContains(t, issueTypes(result.Issues), "insufficient_break")
}

func TestCheckContinuousAndBreaks_ChainStartAfterReport(t *testing.T) {
	// Regression: chainStart must be set to i+1 after reporting consecutive_over_6h,
	// so the next chain doesn't include already-reported slots.
	v1 := vol(1, ts(0, 0), ts(23, 0), 14)
	// 7h chain (triggers report), then 9h break, then 7h chain (should also trigger, not be joined to first)
	day2 := time.Date(2025, 6, 21, 0, 0, 0, 0, utc)
	ts2 := func(h int) time.Time { return time.Date(2025, 6, 21, h, 0, 0, 0, utc) }
	_ = day2

	slots := []Slot{
		// First chain: 7h
		slot(1, ts(8, 0), ts(9, 0), SlotTypeFestival, 1, 1),
		slot(2, ts(9, 0), ts(10, 0), SlotTypeFestival, 1, 1),
		slot(3, ts(10, 0), ts(11, 0), SlotTypeFestival, 1, 1),
		slot(4, ts(11, 0), ts(12, 0), SlotTypeFestival, 1, 1),
		slot(5, ts(12, 0), ts(13, 0), SlotTypeFestival, 1, 1),
		slot(6, ts(13, 0), ts(14, 0), SlotTypeFestival, 1, 1),
		slot(7, ts(14, 0), ts(15, 0), SlotTypeFestival, 1, 1),
		// Break then second chain: 7h next day
		slot(8, ts2(8), ts2(9), SlotTypeFestival, 1, 1),
		slot(9, ts2(9), ts2(10), SlotTypeFestival, 1, 1),
		slot(10, ts2(10), ts2(11), SlotTypeFestival, 1, 1),
		slot(11, ts2(11), ts2(12), SlotTypeFestival, 1, 1),
		slot(12, ts2(12), ts2(13), SlotTypeFestival, 1, 1),
		slot(13, ts2(13), ts2(14), SlotTypeFestival, 1, 1),
		slot(14, ts2(14), ts2(15), SlotTypeFestival, 1, 1),
	}

	result := &ValidationResult{Valid: true}
	checkContinuousAndBreaks(slots, v1, result)

	consecutive := 0
	for _, iss := range result.Issues {
		if iss.Type == "consecutive_over_6h" {
			consecutive++
		}
	}
	// Both chains independently exceed 6h → 2 separate reports
	assert.Equal(t, 2, consecutive, "expected two separate consecutive_over_6h issues, one per chain")
}
