package scheduling

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func makeSchedule(eventStart, festStart, festEnd, eventEnd time.Time) *Schedule {
	return &Schedule{
		ID:            1,
		EventStart:    eventStart,
		FestivalStart: festStart,
		FestivalEnd:   festEnd,
		EventEnd:      eventEnd,
	}
}

func TestGenerateScheduleSlots_Types(t *testing.T) {
	// Event: Wed 18 June (montage) → Festival Thu 19 June 14:00 – Fri 20 June 22:30 → Demontage Sat 21 June
	eventStart := time.Date(2025, 6, 18, 8, 0, 0, 0, warsawLocation)
	festStart := time.Date(2025, 6, 19, 14, 0, 0, 0, warsawLocation)
	festEnd := time.Date(2025, 6, 20, 22, 30, 0, 0, warsawLocation)
	eventEnd := time.Date(2025, 6, 21, 20, 0, 0, 0, warsawLocation)

	schedule := makeSchedule(eventStart, festStart, festEnd, eventEnd)
	slots := generateScheduleSlots(schedule, nil)

	var montage, festival, demontage []Slot
	for _, s := range slots {
		switch s.SlotType {
		case SlotTypeMontage:
			montage = append(montage, s)
		case SlotTypeFestival:
			festival = append(festival, s)
		case SlotTypeDemontage:
			demontage = append(demontage, s)
		}
	}

	// One montage day (18 June, default window 08-20h = 12 hourly slots)
	assert.Len(t, montage, 12)
	assert.Equal(t, 8, montage[0].StartTime.In(warsawLocation).Hour())
	assert.Equal(t, 9, montage[0].EndTime.In(warsawLocation).Hour())
	assert.Equal(t, 1.0, montage[0].CreditHours)
	assert.Equal(t, 2, montage[0].Capacity)
	assert.Equal(t, 20, montage[len(montage)-1].EndTime.In(warsawLocation).Hour())

	// Festival: 14:00 Thu to 22:00 Fri (festEnd 22:30 truncated to 22:00) = 10h Thu + 22h Fri = 32 slots
	assert.Equal(t, 32, len(festival))
	for _, s := range festival {
		assert.Equal(t, 1.0, s.CreditHours)
		assert.Equal(t, 2, s.Capacity)
		dur := s.EndTime.Sub(s.StartTime)
		assert.Equal(t, time.Hour, dur, "each festival slot should be exactly 1h")
	}

	// Demontage: 21 June, default window 08-20h = 12 hourly slots
	assert.Len(t, demontage, 12)
	assert.Equal(t, 1.0, demontage[0].CreditHours)
	assert.Equal(t, 2, demontage[0].Capacity)
	assert.Equal(t, 21, demontage[0].StartTime.In(warsawLocation).Day())
}

func TestGenerateScheduleSlots_FestEndTruncated(t *testing.T) {
	// festEnd at :37 — last slot should end at :00 of that hour, not :37
	festStart := time.Date(2025, 6, 20, 20, 0, 0, 0, warsawLocation)
	festEnd := time.Date(2025, 6, 20, 22, 37, 0, 0, warsawLocation) // truncates to 22:00

	schedule := makeSchedule(festStart, festStart, festEnd, time.Date(2025, 6, 21, 20, 0, 0, 0, warsawLocation))
	slots := generateScheduleSlots(schedule, nil)

	var festivalSlots []Slot
	for _, s := range slots {
		if s.SlotType == SlotTypeFestival {
			festivalSlots = append(festivalSlots, s)
		}
	}

	// 20:00-21:00, 21:00-22:00 — 2 slots (22:37 truncated to 22:00)
	assert.Len(t, festivalSlots, 2)
	last := festivalSlots[len(festivalSlots)-1]
	assert.Equal(t, 22, last.EndTime.In(warsawLocation).Hour())
	assert.Equal(t, 0, last.EndTime.In(warsawLocation).Minute())
}

func TestGenerateScheduleSlots_MultipleMontageDays(t *testing.T) {
	// 3 days of montage before festival
	eventStart := time.Date(2025, 6, 16, 8, 0, 0, 0, warsawLocation)
	festStart := time.Date(2025, 6, 19, 12, 0, 0, 0, warsawLocation)
	festEnd := time.Date(2025, 6, 19, 14, 0, 0, 0, warsawLocation)
	eventEnd := time.Date(2025, 6, 20, 20, 0, 0, 0, warsawLocation)

	schedule := makeSchedule(eventStart, festStart, festEnd, eventEnd)
	slots := generateScheduleSlots(schedule, nil)

	var montage []Slot
	for _, s := range slots {
		if s.SlotType == SlotTypeMontage {
			montage = append(montage, s)
		}
	}

	// 16, 17, 18 June × 12 slots/day (08-20h default) = 36 montage slots
	assert.Equal(t, 3*12, len(montage))
}

func TestGenerateScheduleSlots_CustomDayWindow(t *testing.T) {
	eventStart := time.Date(2025, 6, 18, 8, 0, 0, 0, warsawLocation)
	festStart := time.Date(2025, 6, 19, 10, 0, 0, 0, warsawLocation)
	festEnd := time.Date(2025, 6, 19, 12, 0, 0, 0, warsawLocation)
	eventEnd := time.Date(2025, 6, 20, 20, 0, 0, 0, warsawLocation)

	schedule := makeSchedule(eventStart, festStart, festEnd, eventEnd)

	// Custom window: 18.06 → 10:00-18:00 = 8 slots
	windows := map[string][2]int{"2025-06-18": {10, 18}}
	slots := generateScheduleSlots(schedule, windows)

	var montage []Slot
	for _, s := range slots {
		if s.SlotType == SlotTypeMontage {
			montage = append(montage, s)
		}
	}

	assert.Len(t, montage, 8)
	assert.Equal(t, 10, montage[0].StartTime.In(warsawLocation).Hour())
	assert.Equal(t, 18, montage[len(montage)-1].EndTime.In(warsawLocation).Hour())
}

func TestGenerateScheduleSlots_Labels(t *testing.T) {
	eventStart := time.Date(2025, 6, 18, 8, 0, 0, 0, warsawLocation)
	festStart := time.Date(2025, 6, 19, 10, 0, 0, 0, warsawLocation)
	festEnd := time.Date(2025, 6, 19, 12, 0, 0, 0, warsawLocation)
	eventEnd := time.Date(2025, 6, 20, 20, 0, 0, 0, warsawLocation)

	schedule := makeSchedule(eventStart, festStart, festEnd, eventEnd)
	slots := generateScheduleSlots(schedule, nil)

	for _, s := range slots {
		assert.NotNil(t, s.Label, "all generated slots should have a label")
		assert.NotEmpty(t, *s.Label)
	}
}
