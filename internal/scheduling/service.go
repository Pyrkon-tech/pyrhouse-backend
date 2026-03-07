package scheduling

import (
	"fmt"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateSchedule(req CreateScheduleRequest) (*Schedule, error) {
	return s.repo.CreateSchedule(req)
}

func (s *Service) GetSchedules() ([]Schedule, error) {
	return s.repo.GetSchedules()
}

func (s *Service) GetScheduleDetail(id int) (*ScheduleDetail, error) {
	schedule, err := s.repo.GetSchedule(id)
	if err != nil {
		return nil, err
	}
	if schedule == nil {
		return nil, nil
	}

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

	// Build slot → volunteers map
	slotVolMap := make(map[int][]VolunteerBrief)
	volSlotMap := make(map[int][]int)
	volByID := make(map[int]Volunteer)
	for _, v := range volunteers {
		volByID[v.ID] = v
	}
	for _, a := range assignments {
		if v, ok := volByID[a.VolunteerID]; ok {
			slotVolMap[a.SlotID] = append(slotVolMap[a.SlotID], VolunteerBrief{
				ID:       v.ID,
				Nickname: v.Nickname,
			})
		}
		volSlotMap[a.VolunteerID] = append(volSlotMap[a.VolunteerID], a.SlotID)
	}

	slotsWithVols := make([]SlotWithVolunteers, len(slots))
	for i, sl := range slots {
		vols := slotVolMap[sl.ID]
		if vols == nil {
			vols = []VolunteerBrief{}
		}
		slotsWithVols[i] = SlotWithVolunteers{
			Slot:       sl,
			Volunteers: vols,
		}
	}

	volsWithSlots := make([]VolunteerWithSlots, len(volunteers))
	for i, v := range volunteers {
		sids := volSlotMap[v.ID]
		if sids == nil {
			sids = []int{}
		}
		volsWithSlots[i] = VolunteerWithSlots{
			Volunteer: v,
			SlotIDs:   sids,
		}
	}

	validation := Validate(slots, volunteers, assignments)

	return &ScheduleDetail{
		Schedule:   *schedule,
		Slots:      slotsWithVols,
		Volunteers: volsWithSlots,
		Validation: validation,
	}, nil
}

func (s *Service) ImportVolunteers(scheduleID int, inputs []VolunteerInput) error {
	schedule, err := s.repo.GetSchedule(scheduleID)
	if err != nil {
		return err
	}
	if schedule == nil {
		return fmt.Errorf("schedule not found")
	}

	volunteers := make([]Volunteer, len(inputs))
	for i, input := range inputs {
		availFrom, err := time.Parse("2006-01-02 15:04", input.AvailableFrom)
		if err != nil {
			return fmt.Errorf("invalid available_from for %s: %w", input.Nickname, err)
		}
		availTo, err := time.Parse("2006-01-02 15:04", input.AvailableTo)
		if err != nil {
			return fmt.Errorf("invalid available_to for %s: %w", input.Nickname, err)
		}

		volunteers[i] = Volunteer{
			ScheduleID:    scheduleID,
			UserID:        input.UserID,
			Nickname:      input.Nickname,
			City:          input.City,
			TargetHours:   input.Hours,
			AvailableFrom: availFrom,
			AvailableTo:   availTo,
			Notes:         input.Notes,
		}
	}

	return s.repo.InsertVolunteers(scheduleID, volunteers)
}

func (s *Service) Generate(scheduleID int) (*ScheduleDetail, error) {
	schedule, err := s.repo.GetSchedule(scheduleID)
	if err != nil {
		return nil, err
	}
	if schedule == nil {
		return nil, fmt.Errorf("schedule not found")
	}

	// Clear existing slots and assignments
	if err := s.repo.DeleteAssignments(scheduleID); err != nil {
		return nil, fmt.Errorf("failed to clear assignments: %w", err)
	}
	if err := s.repo.DeleteSlots(scheduleID); err != nil {
		return nil, fmt.Errorf("failed to clear slots: %w", err)
	}

	// Generate slots
	slots := GenerateSlots(schedule)
	if err := s.repo.InsertSlots(slots); err != nil {
		return nil, fmt.Errorf("failed to insert slots: %w", err)
	}

	// Re-read slots (to get IDs)
	slots, err = s.repo.GetSlots(scheduleID)
	if err != nil {
		return nil, err
	}

	// Get volunteers
	volunteers, err := s.repo.GetVolunteers(scheduleID)
	if err != nil {
		return nil, err
	}

	// Solve
	assignments := Solve(slots, volunteers)
	if err := s.repo.InsertAssignments(assignments); err != nil {
		return nil, fmt.Errorf("failed to insert assignments: %w", err)
	}

	// Update assigned hours cache
	assignments, _ = s.repo.GetAssignments(scheduleID)
	slotMap := make(map[int]Slot)
	for _, sl := range slots {
		slotMap[sl.ID] = sl
	}
	volHours := make(map[int]float64)
	for _, a := range assignments {
		if sl, ok := slotMap[a.SlotID]; ok {
			volHours[a.VolunteerID] += sl.CreditHours
		}
	}
	for _, v := range volunteers {
		if err := s.repo.UpdateVolunteerHours(v.ID, volHours[v.ID]); err != nil {
			return nil, err
		}
	}

	return s.GetScheduleDetail(scheduleID)
}

func (s *Service) ValidateSchedule(scheduleID int) (*ValidationResult, error) {
	slots, err := s.repo.GetSlots(scheduleID)
	if err != nil {
		return nil, err
	}
	volunteers, err := s.repo.GetVolunteers(scheduleID)
	if err != nil {
		return nil, err
	}
	assignments, err := s.repo.GetAssignments(scheduleID)
	if err != nil {
		return nil, err
	}

	return Validate(slots, volunteers, assignments), nil
}

func (s *Service) DeleteAssignment(scheduleID, assignmentID int) error {
	assignment, err := s.repo.GetAssignment(assignmentID)
	if err != nil {
		return err
	}
	if assignment == nil {
		return fmt.Errorf("assignment not found")
	}
	return s.repo.DeleteAssignment(assignmentID)
}

func (s *Service) SwapAssignments(scheduleID int, req SwapRequest) error {
	a, err := s.repo.GetAssignment(req.AssignmentA)
	if err != nil {
		return err
	}
	b, err := s.repo.GetAssignment(req.AssignmentB)
	if err != nil {
		return err
	}
	if a == nil || b == nil {
		return fmt.Errorf("one or both assignments not found")
	}

	// Swap: delete both, re-insert with swapped slot IDs
	if err := s.repo.DeleteAssignment(a.ID); err != nil {
		return err
	}
	if err := s.repo.DeleteAssignment(b.ID); err != nil {
		return err
	}

	swapped := []Assignment{
		{SlotID: a.SlotID, VolunteerID: b.VolunteerID},
		{SlotID: b.SlotID, VolunteerID: a.VolunteerID},
	}
	return s.repo.InsertAssignments(swapped)
}

func (s *Service) PublishSchedule(scheduleID int) error {
	schedule, err := s.repo.GetSchedule(scheduleID)
	if err != nil {
		return err
	}
	if schedule == nil {
		return fmt.Errorf("schedule not found")
	}
	return s.repo.UpdateScheduleStatus(scheduleID, "published")
}

func (s *Service) GetSlots(scheduleID int) ([]Slot, error) {
	return s.repo.GetSlots(scheduleID)
}

func (s *Service) GetVolunteers(scheduleID int) ([]Volunteer, error) {
	return s.repo.GetVolunteers(scheduleID)
}

func (s *Service) GetAssignments(scheduleID int) ([]Assignment, error) {
	return s.repo.GetAssignments(scheduleID)
}
