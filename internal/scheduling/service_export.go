package scheduling

import (
	"context"
	"fmt"
	"time"
)

func (s *Service) ExportToSheets() (int, error) {
	if s.sheetsHandler == nil {
		return 0, ErrSheetsUnavailable
	}

	schedule, err := s.getActive()
	if err != nil {
		return 0, err
	}

	ctx := context.Background()
	sheetID, err := s.settingsRepo.Get(ctx, "scheduling.sheet_id")
	if err != nil || sheetID == "" {
		return 0, fmt.Errorf("scheduling.sheet_id not configured in settings")
	}
	sheetName, err := s.settingsRepo.Get(ctx, "scheduling.sheet_name")
	if err != nil || sheetName == "" {
		sheetName = "grafik"
	}

	id := schedule.ID

	slots, err := s.repo.GetSlots(id)
	if err != nil {
		return 0, err
	}
	volunteers, err := s.repo.GetVolunteers(id)
	if err != nil {
		return 0, err
	}
	assignments, err := s.repo.GetAssignments(id)
	if err != nil {
		return 0, err
	}

	return ExportToSheet(s.sheetsHandler, sheetID, sheetName, slots, volunteers, assignments)
}

func (s *Service) ExportCSV() (string, *Schedule, error) {
	schedule, err := s.getActive()
	if err != nil {
		return "", nil, err
	}

	id := schedule.ID

	slots, err := s.repo.GetSlots(id)
	if err != nil {
		return "", nil, err
	}
	volunteers, err := s.repo.GetVolunteers(id)
	if err != nil {
		return "", nil, err
	}
	assignments, err := s.repo.GetAssignments(id)
	if err != nil {
		return "", nil, err
	}

	csv := ExportCSV(schedule, slots, volunteers, assignments)
	return csv, schedule, nil
}

// GetOnDuty returns volunteers on duty at the given time (defaults to now).
func (s *Service) GetOnDuty(at *time.Time) ([]OnDutyEntry, error) {
	t := time.Now()
	if at != nil {
		t = *at
	}
	return s.repo.GetOnDutyVolunteers(t)
}
