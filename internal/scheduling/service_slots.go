package scheduling

import (
	"fmt"
	"time"
)

func (s *Service) CreateSlot(req CreateSlotRequest) (*Slot, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}
	start, err := time.Parse(time.RFC3339, req.Start)
	if err != nil {
		return nil, fmt.Errorf("invalid start time: %w", err)
	}
	end, err := time.Parse(time.RFC3339, req.End)
	if err != nil {
		return nil, fmt.Errorf("invalid end time: %w", err)
	}
	if !end.After(start) {
		return nil, fmt.Errorf("end must be after start")
	}

	creditHours := calculateCreditHours(req.Type, start, end)

	slot := Slot{
		ScheduleID:  schedule.ID,
		SlotType:    req.Type,
		StartTime:   start,
		EndTime:     end,
		CreditHours: creditHours,
		Capacity:    req.Capacity,
		Label:       req.Label,
	}

	return s.repo.CreateSlot(slot)
}

func (s *Service) UpdateSlot(slotID int, req UpdateSlotRequest) (*Slot, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetSlot(slotID)
	if err != nil {
		return nil, err
	}
	if existing == nil || existing.ScheduleID != schedule.ID {
		return nil, ErrSlotNotFound
	}

	if existing.SlotType == SlotTypeFestival && req.Type != nil && *req.Type != SlotTypeFestival {
		return nil, ErrFestivalSlotType
	}

	updates := make(map[string]interface{})

	newStart := existing.StartTime
	newEnd := existing.EndTime
	newType := existing.SlotType

	if req.Start != nil {
		t, err := time.Parse(time.RFC3339, *req.Start)
		if err != nil {
			return nil, fmt.Errorf("invalid start time: %w", err)
		}
		newStart = t
		updates["start_time"] = t
	}
	if req.End != nil {
		t, err := time.Parse(time.RFC3339, *req.End)
		if err != nil {
			return nil, fmt.Errorf("invalid end time: %w", err)
		}
		newEnd = t
		updates["end_time"] = t
	}
	if req.Type != nil {
		newType = *req.Type
		updates["slot_type"] = *req.Type
	}
	if req.Capacity != nil {
		updates["capacity"] = *req.Capacity
	}
	if req.Label != nil {
		updates["label"] = *req.Label
	}

	if req.Start != nil || req.End != nil || req.Type != nil {
		updates["credit_hours"] = calculateCreditHours(newType, newStart, newEnd)
	}

	if len(updates) == 0 {
		return existing, nil
	}

	return s.repo.UpdateSlotByID(slotID, updates)
}

func (s *Service) DeleteSlot(slotID int) error {
	schedule, err := s.getActive()
	if err != nil {
		return err
	}
	existing, err := s.repo.GetSlot(slotID)
	if err != nil {
		return err
	}
	if existing == nil || existing.ScheduleID != schedule.ID {
		return ErrSlotNotFound
	}
	if existing.SlotType == SlotTypeFestival {
		return ErrFestivalSlot
	}

	return s.repo.DeleteSlotByID(slotID)
}
