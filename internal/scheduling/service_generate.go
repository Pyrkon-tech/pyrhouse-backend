package scheduling

import (
	"fmt"
	"time"
	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
)

func (s *Service) Generate() (*ScheduleDetail, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}

	id := schedule.ID

	volunteers, err := s.repo.GetVolunteers(id)
	if err != nil {
		return nil, err
	}

	var blocked []string
	for _, v := range volunteers {
		if v.AssignedHours >= 10 {
			blocked = append(blocked, v.Nickname)
		}
	}
	if len(blocked) > 0 {
		return nil, &GenerateBlockedError{Volunteers: blocked}
	}

	// Clear only assignments, keep slots
	if err := s.repo.DeleteAssignments(id); err != nil {
		return nil, fmt.Errorf("failed to clear assignments: %w", err)
	}

	slots, err := s.repo.GetSlots(id)
	if err != nil {
		return nil, err
	}

	assignments := Solve(slots, volunteers)
	err = repository.WithTransaction(s.repo.DB(), func(tx *goqu.TxDatabase) error {
		if err := s.repo.InsertAssignmentsTx(tx, assignments); err != nil {
			return fmt.Errorf("failed to insert assignments: %w", err)
		}
		return s.repo.RecalcVolunteerHoursTx(tx, id)
	})
	if err != nil {
		return nil, err
	}

	return s.GetScheduleDetail()
}

func (s *Service) ValidateSchedule() (*ValidationResult, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}

	id := schedule.ID

	slots, err := s.repo.GetSlots(id)
	if err != nil {
		return nil, err
	}
	volunteers, err := s.repo.GetVolunteers(id)
	if err != nil {
		return nil, err
	}
	assignments, err := s.repo.GetAssignments(id)
	if err != nil {
		return nil, err
	}

	return Validate(slots, volunteers, assignments), nil
}

// ValidateDraft validates a proposed schedule state without saving it.
func (s *Service) ValidateDraft(req SaveDraftRequest) (*ValidationResult, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}

	volunteers, err := s.repo.GetVolunteers(schedule.ID)
	if err != nil {
		return nil, err
	}

	var slots []Slot
	tempIDCounter := -1
	tempIDMap := make(map[string]int)

	for _, ds := range req.Slots {
		start, err := time.Parse(time.RFC3339, ds.Start)
		if err != nil {
			return nil, fmt.Errorf("invalid start time: %w", err)
		}
		end, err := time.Parse(time.RFC3339, ds.End)
		if err != nil {
			return nil, fmt.Errorf("invalid end time: %w", err)
		}

		slotID := 0
		if ds.ID != nil {
			slotID = *ds.ID
		} else {
			slotID = tempIDCounter
			if ds.TempID != nil {
				tempIDMap[*ds.TempID] = slotID
			}
			tempIDCounter--
		}

		slots = append(slots, Slot{
			ID:          slotID,
			ScheduleID:  schedule.ID,
			SlotType:    ds.Type,
			StartTime:   start,
			EndTime:     end,
			CreditHours: calculateCreditHours(ds.Type, start, end),
			Capacity:    ds.Capacity,
			Label:       ds.Label,
		})
	}

	var assignments []Assignment
	for _, da := range req.Assignments {
		var slotID int
		if da.SlotTempID != nil {
			realID, ok := tempIDMap[*da.SlotTempID]
			if !ok {
				return nil, fmt.Errorf("unknown slot_temp_id: %s", *da.SlotTempID)
			}
			slotID = realID
		} else if da.SlotID != nil {
			slotID = *da.SlotID
		} else {
			return nil, fmt.Errorf("assignment must have slot_id or slot_temp_id")
		}
		assignments = append(assignments, Assignment{
			SlotID:      slotID,
			VolunteerID: da.VolunteerID,
		})
	}

	return Validate(slots, volunteers, assignments), nil
}
