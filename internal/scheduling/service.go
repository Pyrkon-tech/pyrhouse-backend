package scheduling

import (
	"context"
	"fmt"
	"time"
	"warehouse/internal/integrations/googlesheets"
	"warehouse/internal/settings"
)

type Service struct {
	repo          *Repository
	sheetsHandler *googlesheets.GoogleSheetsHandler // may be nil
	settingsRepo  *settings.Repository
}

func NewService(repo *Repository, sheetsHandler *googlesheets.GoogleSheetsHandler, settingsRepo *settings.Repository) *Service {
	return &Service{repo: repo, sheetsHandler: sheetsHandler, settingsRepo: settingsRepo}
}

// getActive returns the active schedule or an error if none exists.
func (s *Service) getActive() (*Schedule, error) {
	schedule, err := s.repo.GetActiveSchedule()
	if err != nil {
		return nil, err
	}
	if schedule == nil {
		return nil, fmt.Errorf("no active schedule")
	}
	return schedule, nil
}

func (s *Service) CreateSchedule(req CreateScheduleRequest) (*Schedule, error) {
	// Archive any existing active schedule
	if err := s.repo.ArchiveAllSchedules(); err != nil {
		return nil, fmt.Errorf("failed to archive previous schedule: %w", err)
	}
	return s.repo.CreateSchedule(req)
}

func (s *Service) GetScheduleDetail() (*ScheduleDetail, error) {
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

func (s *Service) ImportVolunteers(inputs []VolunteerInput) error {
	schedule, err := s.getActive()
	if err != nil {
		return err
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
			ScheduleID:    schedule.ID,
			UserID:        input.UserID,
			Nickname:      input.Nickname,
			City:          input.City,
			TargetHours:   input.Hours,
			AvailableFrom: availFrom,
			AvailableTo:   availTo,
			Notes:         input.Notes,
		}
	}

	return s.repo.InsertVolunteers(schedule.ID, volunteers)
}

// ImportVolunteersFromSheet reads volunteer data from a Google Spreadsheet and imports them.
// Expected columns: Pseudonim, Miasto, Godziny, Dostępny od, Dostępny do, Uwagi
func (s *Service) ImportVolunteersFromSheet(req ImportFromSheetRequest) (int, error) {
	if s.sheetsHandler == nil {
		return 0, fmt.Errorf("Google Sheets integration not available")
	}

	schedule, err := s.getActive()
	if err != nil {
		return 0, err
	}

	readRange := fmt.Sprintf("%s!A1:Z999", req.SheetName)
	rows, err := s.sheetsHandler.ReadSpreadsheet(req.SheetID, readRange)
	if err != nil {
		return 0, fmt.Errorf("failed to read sheet: %w", err)
	}

	if len(rows) < 2 {
		return 0, fmt.Errorf("sheet has no data rows (only header or empty)")
	}

	cols, err := parseHeader(rows[0])
	if err != nil {
		return 0, fmt.Errorf("invalid sheet header: %w", err)
	}

	// Parse event date range for day-name resolution
	eventStart, err := time.Parse("2006-01-02", schedule.EventStart)
	if err != nil {
		return 0, fmt.Errorf("invalid event_start date: %w", err)
	}
	eventEnd, err := time.Parse("2006-01-02", schedule.EventEnd)
	if err != nil {
		return 0, fmt.Errorf("invalid event_end date: %w", err)
	}

	var volunteers []Volunteer
	var skipped int
	for i, row := range rows[1:] {
		nickname := cellStr(row, cols.nickname)
		if nickname == "" {
			continue
		}

		if shouldSkipByTag(row, cols.tags) {
			skipped++
			continue
		}

		hours := cellInt(row, cols.hours, 14)
		city := cellStrPtr(row, cols.city)
		notes := cellStrPtr(row, cols.notes)

		availFromStr := cellStr(row, cols.availableFrom)
		availToStr := cellStr(row, cols.availableTo)

		availFrom, err := parsePolishDayTime(availFromStr, eventStart, eventEnd)
		if err != nil {
			return 0, fmt.Errorf("row %d (%s): invalid '%s': %w", i+2, nickname, colAvailableFrom, err)
		}
		availTo, err := parsePolishDayTime(availToStr, eventStart, eventEnd)
		if err != nil {
			return 0, fmt.Errorf("row %d (%s): invalid '%s': %w", i+2, nickname, colAvailableTo, err)
		}

		volunteers = append(volunteers, Volunteer{
			ScheduleID:    schedule.ID,
			Nickname:      nickname,
			City:          city,
			TargetHours:   hours,
			AvailableFrom: availFrom,
			AvailableTo:   availTo,
			Notes:         notes,
		})
	}

	_ = skipped

	if len(volunteers) == 0 {
		return 0, fmt.Errorf("no valid volunteer rows found in sheet")
	}

	if err := s.repo.InsertVolunteers(schedule.ID, volunteers); err != nil {
		return 0, err
	}

	return len(volunteers), nil
}

func (s *Service) Generate() (*ScheduleDetail, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}

	id := schedule.ID

	// Clear existing slots and assignments
	if err := s.repo.DeleteAssignments(id); err != nil {
		return nil, fmt.Errorf("failed to clear assignments: %w", err)
	}
	if err := s.repo.DeleteSlots(id); err != nil {
		return nil, fmt.Errorf("failed to clear slots: %w", err)
	}

	// Generate slots
	slots := GenerateSlots(schedule)
	if err := s.repo.InsertSlots(slots); err != nil {
		return nil, fmt.Errorf("failed to insert slots: %w", err)
	}

	// Re-read slots (to get IDs)
	slots, err = s.repo.GetSlots(id)
	if err != nil {
		return nil, err
	}

	// Get volunteers
	volunteers, err := s.repo.GetVolunteers(id)
	if err != nil {
		return nil, err
	}

	// Solve
	assignments := Solve(slots, volunteers)
	if err := s.repo.InsertAssignments(assignments); err != nil {
		return nil, fmt.Errorf("failed to insert assignments: %w", err)
	}

	// Update assigned hours cache
	assignments, _ = s.repo.GetAssignments(id)
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

func (s *Service) DeleteAssignment(assignmentID int) error {
	assignment, err := s.repo.GetAssignment(assignmentID)
	if err != nil {
		return err
	}
	if assignment == nil {
		return fmt.Errorf("assignment not found")
	}
	return s.repo.DeleteAssignment(assignmentID)
}

func (s *Service) SwapAssignments(req SwapRequest) error {
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

func (s *Service) PublishSchedule() error {
	schedule, err := s.getActive()
	if err != nil {
		return err
	}
	return s.repo.UpdateScheduleStatus(schedule.ID, "published")
}

func (s *Service) UpdateVolunteer(volunteerID int, req UpdateVolunteerRequest) (*Volunteer, error) {
	updates := make(map[string]interface{})

	if req.Nickname != nil {
		updates["nickname"] = *req.Nickname
	}
	if req.City != nil {
		updates["city"] = *req.City
	}
	if req.Hours != nil {
		updates["target_hours"] = *req.Hours
	}
	if req.AvailableFrom != nil {
		t, err := time.Parse("2006-01-02 15:04", *req.AvailableFrom)
		if err != nil {
			return nil, fmt.Errorf("invalid available_from: %w", err)
		}
		updates["available_from"] = t
	}
	if req.AvailableTo != nil {
		t, err := time.Parse("2006-01-02 15:04", *req.AvailableTo)
		if err != nil {
			return nil, fmt.Errorf("invalid available_to: %w", err)
		}
		updates["available_to"] = t
	}
	if req.Notes != nil {
		updates["notes"] = *req.Notes
	}
	if req.UserID != nil {
		updates["user_id"] = *req.UserID
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	return s.repo.UpdateVolunteer(volunteerID, updates)
}

func (s *Service) GetVolunteers() ([]Volunteer, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}
	return s.repo.GetVolunteers(schedule.ID)
}

func (s *Service) ExportToSheets() (int, error) {
	if s.sheetsHandler == nil {
		return 0, fmt.Errorf("Google Sheets integration not available")
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
