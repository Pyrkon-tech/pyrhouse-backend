package scheduling

import (
	"fmt"
	"time"
)

type VolunteerSchedule struct {
	Volunteer *Volunteer      `json:"volunteer"`
	Slots     []VolunteerSlot `json:"slots"`
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
		return nil, ErrSheetsUnavailable
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

func (s *Service) DeleteVolunteer(volunteerID int) (bool, error) {
	return s.repo.DeleteVolunteer(volunteerID)
}

func (s *Service) GetVolunteers() ([]Volunteer, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}
	return s.repo.GetVolunteers(schedule.ID)
}

func (s *Service) GetMySchedule(userID int) (*VolunteerSchedule, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}
	volunteer, err := s.repo.GetVolunteerByUserID(schedule.ID, userID)
	if err != nil {
		return nil, err
	}
	if volunteer == nil {
		return nil, nil
	}
	slots, err := s.repo.GetSlotsByVolunteer(volunteer.ID)
	if err != nil {
		return nil, err
	}
	if slots == nil {
		slots = []VolunteerSlot{}
	}
	return &VolunteerSchedule{Volunteer: volunteer, Slots: slots}, nil
}
