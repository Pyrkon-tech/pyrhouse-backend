package scheduling

import (
	"context"
	"fmt"
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

	eventStart := schedule.EventStart
	eventEnd := schedule.EventEnd

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

	// Generate slots (pass volunteer count for dynamic capacity)
	volunteers, err := s.repo.GetVolunteers(id)
	if err != nil {
		return nil, err
	}
	slots := GenerateSlots(schedule, len(volunteers))
	if err := s.repo.InsertSlots(slots); err != nil {
		return nil, fmt.Errorf("failed to insert slots: %w", err)
	}

	// Re-read slots (to get IDs)
	slots, err = s.repo.GetSlots(id)
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

func (s *Service) AddAssignment(req AddAssignmentRequest) (*Assignment, error) {
	assignment, err := s.repo.CreateAssignment(req.SlotID, req.VolunteerID)
	if err != nil {
		return nil, err
	}
	return assignment, nil
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

	// Block publish if there are error-severity validation issues
	slots, err := s.repo.GetSlots(schedule.ID)
	if err != nil {
		return err
	}
	volunteers, err := s.repo.GetVolunteers(schedule.ID)
	if err != nil {
		return err
	}
	assignments, err := s.repo.GetAssignments(schedule.ID)
	if err != nil {
		return err
	}
	result := Validate(slots, volunteers, assignments)
	for _, issue := range result.Issues {
		if issue.Severity == "error" {
			return fmt.Errorf("cannot publish: schedule has validation errors")
		}
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

	// Recalc credit_hours if time or type changed
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
		existingSlots, err := s.repo.GetSlots(schedule.ID)
		if err != nil {
			return err
		}

		existingIDs := make(map[int]bool)
		for _, sl := range existingSlots {
			existingIDs[sl.ID] = true
		}

		payloadIDs := make(map[int]bool)
		tempIDMap := make(map[string]int)

		// 1. Process slots: update existing, insert new
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
				// UPDATE existing slot
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
				// INSERT new slot
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

		// 2. Delete slots in DB but NOT in payload (CASCADE deletes assignments)
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

		// 3. Resolve assignments: replace temp_id with real id
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

		// 4. Replace all assignments
		if err := s.repo.DeleteAssignmentsByScheduleTx(tx, schedule.ID); err != nil {
			return err
		}
		if err := s.repo.InsertAssignmentsTx(tx, resolved); err != nil {
			return err
		}

		// 5. Recalc volunteer hours
		if err := s.repo.RecalcVolunteerHoursTx(tx, schedule.ID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Build response (outside transaction)
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

	// Build slots from draft data (in-memory, no DB)
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

	// Build assignments
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
