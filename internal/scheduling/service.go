package scheduling

import (
	"context"
	"fmt"
	"strings"
	"time"
	"warehouse/internal/integrations/googlesheets"
	"warehouse/internal/repository"
	"warehouse/internal/settings"

	"github.com/doug-martin/goqu/v9"
)

type Service struct {
	repo          *Repository
	sheetsHandler *googlesheets.GoogleSheetsHandler // may be nil
	settingsRepo  *settings.Repository
}

func NewService(repo *Repository, sheetsHandler *googlesheets.GoogleSheetsHandler, settingsRepo *settings.Repository) *Service {
	return &Service{repo: repo, sheetsHandler: sheetsHandler, settingsRepo: settingsRepo}
}

func (s *Service) publishedGuard(schedule *Schedule) error {
	if schedule.Status == "published" {
		return ErrSchedulePublished
	}
	return nil
}

func calculateCreditHours(slotType string, start, end time.Time) float64 {
	if slotType == SlotTypeMontage || slotType == SlotTypeDemontage {
		return 7
	}
	return end.Sub(start).Hours()
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

func (s *Service) CreateSchedule(req CreateScheduleRequest) (*ScheduleDetail, error) {
	if err := s.repo.ArchiveAllSchedules(); err != nil {
		return nil, fmt.Errorf("failed to archive previous schedule: %w", err)
	}

	schedule, err := s.repo.CreateSchedule(req)
	if err != nil {
		return nil, err
	}

	// Auto-generate festival slots (hourly blocks between festival_start and festival_end)
	festivalSlots := generateFestivalSlots(schedule)
	if len(festivalSlots) > 0 {
		if _, err := s.repo.InsertSlotsReturning(festivalSlots); err != nil {
			return nil, fmt.Errorf("failed to insert festival slots: %w", err)
		}
	}

	return s.GetScheduleDetail()
}

// generateFestivalSlots creates 1-hour blocks covering festival_start → festival_end.
func generateFestivalSlots(schedule *Schedule) []Slot {
	var slots []Slot
	loc := schedule.FestivalStart.Location()
	cur := schedule.FestivalStart.Truncate(time.Hour)
	end := schedule.FestivalEnd

	for cur.Before(end) {
		slotEnd := cur.Add(time.Hour)
		if slotEnd.After(end) {
			slotEnd = end
		}
		label := fmt.Sprintf("Festiwal %s", cur.Format("02.01 15:04"))
		labelStr := label
		slots = append(slots, Slot{
			ScheduleID:  schedule.ID,
			SlotType:    SlotTypeFestival,
			StartTime:   cur.In(loc),
			EndTime:     slotEnd.In(loc),
			CreditHours: slotEnd.Sub(cur).Hours(),
			Capacity:    2,
			Label:       &labelStr,
		})
		cur = slotEnd
	}
	return slots
}

func (s *Service) GetScheduleDetail() (*ScheduleDetail, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}
	return s.buildDetail(schedule)
}

func (s *Service) buildDetail(schedule *Schedule) (*ScheduleDetail, error) {
	id := schedule.ID

	slots, err := s.repo.GetSlots(id)
	if err != nil {
		return nil, err
	}

	volunteers, err := s.repo.GetVolunteers(id)
	if err != nil {
		return nil, err
	}

	assignments, err := s.repo.GetAssignmentsWithNicknames(id)
	if err != nil {
		return nil, err
	}

	// Build slot → SlotVolunteer map (using assignment_id as ID)
	slotVolMap := make(map[int][]SlotVolunteer)
	volSlotMap := make(map[int][]int)
	for _, a := range assignments {
		slotVolMap[a.SlotID] = append(slotVolMap[a.SlotID], SlotVolunteer{
			ID:          a.ID, // assignment_id
			VolunteerID: a.VolunteerID,
			Nickname:    a.Nickname,
		})
		volSlotMap[a.VolunteerID] = append(volSlotMap[a.VolunteerID], a.SlotID)
	}

	slotsWithVols := make([]SlotWithVolunteers, len(slots))
	for i, sl := range slots {
		vols := slotVolMap[sl.ID]
		if vols == nil {
			vols = []SlotVolunteer{}
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

	// GetAssignments (without nicknames) for validator
	plainAssignments, err := s.repo.GetAssignments(id)
	if err != nil {
		return nil, err
	}
	validation := Validate(slots, volunteers, plainAssignments)

	return &ScheduleDetail{
		Schedule:   *schedule,
		Slots:      slotsWithVols,
		Volunteers: volsWithSlots,
		Validation: validation,
	}, nil
}

func (s *Service) ImportVolunteers(inputs []VolunteerInput) (*ImportResult, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.GetVolunteers(schedule.ID)
	if err != nil {
		return nil, err
	}
	existingByNick := make(map[string]bool, len(existing))
	for _, v := range existing {
		existingByNick[v.Nickname] = true
	}

	var toInsert []Volunteer
	updated := 0
	for _, input := range inputs {
		availFrom, err := time.ParseInLocation("2006-01-02 15:04", input.AvailableFrom, warsawLocation)
		if err != nil {
			return nil, fmt.Errorf("invalid available_from for %s: %w", input.Nickname, err)
		}
		availTo, err := time.ParseInLocation("2006-01-02 15:04", input.AvailableTo, warsawLocation)
		if err != nil {
			return nil, fmt.Errorf("invalid available_to for %s: %w", input.Nickname, err)
		}

		if existingByNick[input.Nickname] {
			updated++
			continue
		}

		toInsert = append(toInsert, Volunteer{
			ScheduleID:    schedule.ID,
			UserID:        input.UserID,
			Nickname:      input.Nickname,
			City:          input.City,
			TargetHours:   input.Hours,
			AvailableFrom: availFrom,
			AvailableTo:   availTo,
			Notes:         input.Notes,
		})
	}

	if len(toInsert) > 0 {
		if err := s.repo.InsertVolunteers(schedule.ID, toInsert); err != nil {
			return nil, err
		}
	}

	return &ImportResult{
		Imported: len(toInsert),
		Updated:  updated,
		Skipped:  0,
	}, nil
}

// ImportVolunteersFromSheet reads volunteer data from a Google Spreadsheet and imports them.
// Expected columns: Pseudonim, Miasto, Godziny, Dostępny od, Dostępny do, Uwagi
func (s *Service) ImportVolunteersFromSheet(req ImportFromSheetRequest) (*ImportSheetResult, error) {
	if s.sheetsHandler == nil {
		return nil, fmt.Errorf("Google Sheets integration not available")
	}

	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}

	readRange := fmt.Sprintf("%s!A1:Z999", req.SheetName)
	rows, err := s.sheetsHandler.ReadSpreadsheet(req.SheetID, readRange)
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("sheet has no data rows (only header or empty)")
	}

	cols, err := parseHeader(rows[0])
	if err != nil {
		return nil, fmt.Errorf("invalid sheet header: %w", err)
	}

	eventStart := schedule.EventStart
	eventEnd := schedule.EventEnd

	existing, err := s.repo.GetVolunteers(schedule.ID)
	if err != nil {
		return nil, err
	}
	existingByNick := make(map[string]bool, len(existing))
	for _, v := range existing {
		existingByNick[v.Nickname] = true
	}

	var volunteers []Volunteer
	var parseErrors []string
	skipped := 0
	updated := 0

	for i, row := range rows[1:] {
		nickname := cellStr(row, cols.nickname)
		if nickname == "" {
			continue
		}

		if shouldSkipByTag(row, cols.tags) {
			skipped++
			continue
		}

		if existingByNick[nickname] {
			updated++
			continue
		}

		hours := cellInt(row, cols.hours, 14)
		city := cellStrPtr(row, cols.city)
		notes := cellStrPtr(row, cols.notes)

		availFromStr := cellStr(row, cols.availableFrom)
		availToStr := cellStr(row, cols.availableTo)

		availFrom, err := parsePolishDayTime(availFromStr, eventStart, eventEnd)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("row %d (%s): invalid '%s': %v", i+2, nickname, colAvailableFrom, err))
			continue
		}
		availTo, err := parsePolishDayTime(availToStr, eventStart, eventEnd)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("row %d (%s): invalid '%s': %v", i+2, nickname, colAvailableTo, err))
			continue
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

	if len(volunteers) > 0 {
		if err := s.repo.InsertVolunteers(schedule.ID, volunteers); err != nil {
			return nil, err
		}
	}

	return &ImportSheetResult{
		Imported: len(volunteers),
		Updated:  updated,
		Skipped:  skipped,
		Errors:   parseErrors,
	}, nil
}

func (s *Service) Generate() (*ScheduleDetail, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}

	id := schedule.ID

	// Clear only assignments, keep slots
	if err := s.repo.DeleteAssignments(id); err != nil {
		return nil, fmt.Errorf("failed to clear assignments: %w", err)
	}

	volunteers, err := s.repo.GetVolunteers(id)
	if err != nil {
		return nil, err
	}
	slots, err := s.repo.GetSlots(id)
	if err != nil {
		return nil, err
	}

	assignments := Solve(slots, volunteers)
	if err := s.repo.InsertAssignments(assignments); err != nil {
		return nil, fmt.Errorf("failed to insert assignments: %w", err)
	}

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

func (s *Service) AddAssignment(req AddAssignmentRequest) (*AssignmentDetail, error) {
	assignment, err := s.repo.CreateAssignment(req.SlotID, req.VolunteerID)
	if err != nil {
		if isDuplicateError(err) {
			return nil, &DuplicateAssignmentError{VolunteerID: req.VolunteerID, SlotID: req.SlotID}
		}
		return nil, err
	}

	row, err := s.repo.GetAssignmentWithNickname(assignment.ID)
	if err != nil || row == nil {
		return &AssignmentDetail{
			ID:          assignment.ID,
			SlotID:      assignment.SlotID,
			VolunteerID: assignment.VolunteerID,
		}, nil
	}
	return &AssignmentDetail{
		ID:          row.ID,
		SlotID:      row.SlotID,
		VolunteerID: row.VolunteerID,
		Nickname:    row.Nickname,
	}, nil
}

type DuplicateAssignmentError struct {
	VolunteerID int
	SlotID      int
}

func (e *DuplicateAssignmentError) Error() string {
	return "already_assigned"
}

func isDuplicateError(err error) bool {
	return strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "unique constraint") ||
		strings.Contains(err.Error(), "UNIQUE constraint")
}

func (s *Service) DeleteAssignment(assignmentID int) error {
	exists, err := s.repo.AssignmentExists(assignmentID)
	if err != nil {
		return err
	}
	if !exists {
		return nil // idempotent — already gone
	}
	return s.repo.DeleteAssignment(assignmentID)
}

func (s *Service) MoveAssignment(req MoveAssignmentRequest) (*MoveResponse, error) {
	a, err := s.repo.GetAssignmentWithNickname(req.AssignmentID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, fmt.Errorf("assignment not found")
	}

	deletedID := a.ID

	// Delete old assignment, create new one in a transaction
	var newAssignment Assignment
	err = repository.WithTransaction(s.repo.DB(), func(tx *goqu.TxDatabase) error {
		if err := s.repo.DeleteAssignmentTx(tx, deletedID); err != nil {
			return err
		}
		created, err := s.repo.CreateAssignmentTx(tx, req.ToSlotID, a.VolunteerID)
		if err != nil {
			if isDuplicateError(err) {
				return &DuplicateAssignmentError{VolunteerID: a.VolunteerID, SlotID: req.ToSlotID}
			}
			return err
		}
		newAssignment = *created
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &MoveResponse{
		DeletedAssignmentID: deletedID,
		CreatedAssignment: AssignmentDetail{
			ID:          newAssignment.ID,
			SlotID:      newAssignment.SlotID,
			VolunteerID: newAssignment.VolunteerID,
			Nickname:    a.Nickname,
		},
	}, nil
}

func (s *Service) SwapAssignments(req SwapRequest) (*SwapResponse, error) {
	a, err := s.repo.GetAssignmentWithNickname(req.AssignmentA)
	if err != nil {
		return nil, err
	}
	b, err := s.repo.GetAssignmentWithNickname(req.AssignmentB)
	if err != nil {
		return nil, err
	}
	if a == nil || b == nil {
		return nil, fmt.Errorf("one or both assignments not found")
	}

	var newA, newB Assignment
	err = repository.WithTransaction(s.repo.DB(), func(tx *goqu.TxDatabase) error {
		if err := s.repo.DeleteAssignmentTx(tx, a.ID); err != nil {
			return err
		}
		if err := s.repo.DeleteAssignmentTx(tx, b.ID); err != nil {
			return err
		}
		created, err := s.repo.CreateAssignmentTx(tx, b.SlotID, a.VolunteerID)
		if err != nil {
			return err
		}
		newA = *created
		created, err = s.repo.CreateAssignmentTx(tx, a.SlotID, b.VolunteerID)
		if err != nil {
			return err
		}
		newB = *created
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &SwapResponse{
		AssignmentA: AssignmentDetail{ID: newA.ID, SlotID: newA.SlotID, VolunteerID: newA.VolunteerID, Nickname: a.Nickname},
		AssignmentB: AssignmentDetail{ID: newB.ID, SlotID: newB.SlotID, VolunteerID: newB.VolunteerID, Nickname: b.Nickname},
	}, nil
}

func (s *Service) ChangeStatus(newStatus string) (*ScheduleDetail, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}

	if newStatus == "published" {
		slots, err := s.repo.GetSlots(schedule.ID)
		if err != nil {
			return nil, err
		}
		volunteers, err := s.repo.GetVolunteers(schedule.ID)
		if err != nil {
			return nil, err
		}
		assignments, err := s.repo.GetAssignments(schedule.ID)
		if err != nil {
			return nil, err
		}
		result := Validate(slots, volunteers, assignments)
		for _, issue := range result.Issues {
			if issue.Severity == "error" {
				return nil, &ValidationBlockedError{ErrorCount: countErrors(result.Issues)}
			}
		}
	}

	dbStatus := newStatus
	if newStatus == "draft" {
		dbStatus = "active"
	}

	updated, err := s.repo.UpdateScheduleStatus(schedule.ID, dbStatus)
	if err != nil {
		return nil, err
	}

	return s.buildDetail(updated)
}

type ValidationBlockedError struct {
	ErrorCount int
}

func (e *ValidationBlockedError) Error() string {
	return "validation_failed"
}

func countErrors(issues []ValidationIssue) int {
	n := 0
	for _, i := range issues {
		if i.Severity == "error" {
			n++
		}
	}
	return n
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
		t, err := time.ParseInLocation("2006-01-02 15:04", *req.AvailableFrom, warsawLocation)
		if err != nil {
			return nil, fmt.Errorf("invalid available_from: %w", err)
		}
		updates["available_from"] = t
	}
	if req.AvailableTo != nil {
		t, err := time.ParseInLocation("2006-01-02 15:04", *req.AvailableTo, warsawLocation)
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

// Slot CRUD

func (s *Service) CreateSlot(req CreateSlotRequest) (*Slot, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}
	if err := s.publishedGuard(schedule); err != nil {
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
	if err := s.publishedGuard(schedule); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetSlot(slotID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("slot not found")
	}

	if existing.SlotType == SlotTypeFestival && req.Type != nil && *req.Type != SlotTypeFestival {
		return nil, fmt.Errorf("cannot change type of festival slot")
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
	if err := s.publishedGuard(schedule); err != nil {
		return err
	}

	existing, err := s.repo.GetSlot(slotID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("slot not found")
	}
	if existing.SlotType == SlotTypeFestival {
		return fmt.Errorf("festival slot")
	}

	return s.repo.DeleteSlotByID(slotID)
}

// SaveDraft performs a bulk save of the entire schedule state in a single transaction.
func (s *Service) SaveDraft(req SaveDraftRequest) (*SaveDraftResponse, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}
	if err := s.publishedGuard(schedule); err != nil {
		return nil, err
	}

	var createdSlots []TempIDMapping

	err = repository.WithTransaction(s.repo.DB(), func(tx *goqu.TxDatabase) error {
		// Optimistic locking: bump version only if client version matches
		if req.Version > 0 {
			ok, _, err := s.repo.BumpVersionIfMatchTx(tx, schedule.ID, req.Version)
			if err != nil {
				return err
			}
			if !ok {
				return &VersionConflictError{ServerVersion: schedule.Version, YourVersion: req.Version}
			}
		}

		existingSlots, err := s.repo.GetSlots(schedule.ID)
		if err != nil {
			return err
		}

		// Separate festival slots (protected) from editable ones
		festivalIDs := make(map[int]bool)
		existingIDs := make(map[int]bool)
		for _, sl := range existingSlots {
			if sl.SlotType == SlotTypeFestival {
				festivalIDs[sl.ID] = true
			} else {
				existingIDs[sl.ID] = true
			}
		}

		payloadIDs := make(map[int]bool)
		tempIDMap := make(map[string]int)

		for _, ds := range req.Slots {
			start, err := time.Parse(time.RFC3339, ds.Start)
			if err != nil {
				return fmt.Errorf("invalid start time: %w", err)
			}
			end, err := time.Parse(time.RFC3339, ds.End)
			if err != nil {
				return fmt.Errorf("invalid end time: %w", err)
			}
			creditHours := calculateCreditHours(ds.Type, start, end)

			if ds.ID != nil {
				if festivalIDs[*ds.ID] {
					// festival slot in payload — skip silently (protected)
					payloadIDs[*ds.ID] = true
					continue
				}
				record := goqu.Record{
					"slot_type":    ds.Type,
					"start_time":   start,
					"end_time":     end,
					"credit_hours": creditHours,
					"capacity":     ds.Capacity,
					"label":        ds.Label,
				}
				if err := s.repo.UpdateSlotTx(tx, *ds.ID, record); err != nil {
					return err
				}
				payloadIDs[*ds.ID] = true
			} else {
				// INSERT new slot — only montage/demontage allowed via draft
				if ds.Type == SlotTypeFestival {
					return fmt.Errorf("cannot create festival slots via draft")
				}
				newSlot, err := s.repo.InsertSlotTx(tx, Slot{
					ScheduleID:  schedule.ID,
					SlotType:    ds.Type,
					StartTime:   start,
					EndTime:     end,
					CreditHours: creditHours,
					Capacity:    ds.Capacity,
					Label:       ds.Label,
				})
				if err != nil {
					return err
				}
				if ds.TempID != nil {
					tempIDMap[*ds.TempID] = newSlot.ID
					createdSlots = append(createdSlots, TempIDMapping{
						TempID: *ds.TempID,
						ID:     newSlot.ID,
					})
				}
			}
		}

		// Delete non-festival slots absent from payload
		var toDelete []int
		for id := range existingIDs {
			if !payloadIDs[id] {
				toDelete = append(toDelete, id)
			}
		}
		if len(toDelete) > 0 {
			if err := s.repo.DeleteSlotsByIDsTx(tx, toDelete); err != nil {
				return err
			}
		}

		// Resolve assignments: replace temp_id with real id
		var resolved []Assignment
		for _, da := range req.Assignments {
			var slotID int
			if da.SlotTempID != nil {
				realID, ok := tempIDMap[*da.SlotTempID]
				if !ok {
					return fmt.Errorf("unknown slot_temp_id: %s", *da.SlotTempID)
				}
				slotID = realID
			} else if da.SlotID != nil {
				slotID = *da.SlotID
			} else {
				return fmt.Errorf("assignment must have slot_id or slot_temp_id")
			}
			resolved = append(resolved, Assignment{
				SlotID:      slotID,
				VolunteerID: da.VolunteerID,
			})
		}

		if err := s.repo.DeleteAssignmentsByScheduleTx(tx, schedule.ID); err != nil {
			return err
		}
		if err := s.repo.InsertAssignmentsTx(tx, resolved); err != nil {
			return err
		}
		if err := s.repo.RecalcVolunteerHoursTx(tx, schedule.ID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	detail, err := s.GetScheduleDetail()
	if err != nil {
		return nil, err
	}

	return &SaveDraftResponse{
		Schedule:     *detail,
		CreatedSlots: createdSlots,
		Validation:   detail.Validation,
	}, nil
}

type VersionConflictError struct {
	ServerVersion int
	YourVersion   int
}

func (e *VersionConflictError) Error() string {
	return "version_conflict"
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
