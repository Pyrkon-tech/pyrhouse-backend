package scheduling

import (
	"fmt"
	"time"
	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
)

func (s *Service) CreateSchedule(req CreateScheduleRequest) (*ScheduleDetail, error) {
	var schedule *Schedule
	err := repository.WithTransaction(s.repo.DB(), func(tx *goqu.TxDatabase) error {
		if err := s.repo.ArchiveAllSchedulesTx(tx); err != nil {
			return fmt.Errorf("failed to archive previous schedule: %w", err)
		}
		var err error
		schedule, err = s.repo.CreateScheduleTx(tx, req)
		if err != nil {
			return err
		}
		// No day windows yet for a brand-new schedule — defaults (8-20h) apply.
		slots := generateScheduleSlots(schedule, nil)
		return s.repo.InsertSlotsReturningTx(tx, slots)
	})
	if err != nil {
		return nil, err
	}
	return s.GetScheduleDetail()
}

// RegenerateSlots deletes all non-festival slots for the active schedule, then
// re-creates montage/demontage slots according to the current day windows.
// Existing assignments are cleared (festival slots and their assignments are preserved).
func (s *Service) RegenerateSlots() (*ScheduleDetail, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}

	dayWindows, err := s.repo.GetDayWindowsAsMap(schedule.ID)
	if err != nil {
		return nil, err
	}

	err = repository.WithTransaction(s.repo.DB(), func(tx *goqu.TxDatabase) error {
		if err := s.repo.DeleteNonFestivalSlotsTx(tx, schedule.ID); err != nil {
			return err
		}
		slots := generateScheduleSlots(schedule, dayWindows)
		var nonFestival []Slot
		for _, sl := range slots {
			if sl.SlotType != SlotTypeFestival {
				nonFestival = append(nonFestival, sl)
			}
		}
		if err := s.repo.InsertSlotsReturningTx(tx, nonFestival); err != nil {
			return err
		}
		if err := s.repo.RecalcVolunteerHoursTx(tx, schedule.ID); err != nil {
			return err
		}
		_, err := s.repo.IncrementVersionTx(tx, schedule.ID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.GetScheduleDetail()
}

// generateScheduleSlots creates montage (hourly within day window), festival (1h blocks),
// and demontage (hourly within day window) slots. All times in Europe/Warsaw.
// dayWindows maps date string "2006-01-02" → [startHour, endHour]; nil/missing entry → default 8-20.
func generateScheduleSlots(schedule *Schedule, dayWindows map[string][2]int) []Slot {
	var slots []Slot

	windowFor := func(d time.Time) (int, int) {
		key := d.Format("2006-01-02")
		if dayWindows != nil {
			if w, ok := dayWindows[key]; ok {
				return w[0], w[1]
			}
		}
		return 8, 20
	}

	dayAbbr := map[time.Weekday]string{
		time.Monday: "Pn", time.Tuesday: "Wt", time.Wednesday: "Śr",
		time.Thursday: "Czw", time.Friday: "Pt", time.Saturday: "Sb", time.Sunday: "Nd",
	}

	// Montage: event_start (inclusive) up to the day before festival_start, hourly.
	festivalDay := time.Date(
		schedule.FestivalStart.In(warsawLocation).Year(),
		schedule.FestivalStart.In(warsawLocation).Month(),
		schedule.FestivalStart.In(warsawLocation).Day(),
		0, 0, 0, 0, warsawLocation,
	)
	for d := time.Date(
		schedule.EventStart.In(warsawLocation).Year(),
		schedule.EventStart.In(warsawLocation).Month(),
		schedule.EventStart.In(warsawLocation).Day(),
		0, 0, 0, 0, warsawLocation,
	); d.Before(festivalDay); d = d.AddDate(0, 0, 1) {
		startH, endH := windowFor(d)
		abbr := dayAbbr[d.Weekday()]
		for h := startH; h < endH; h++ {
			slotStart := time.Date(d.Year(), d.Month(), d.Day(), h, 0, 0, 0, warsawLocation)
			slotEnd := slotStart.Add(time.Hour)
			label := fmt.Sprintf("Montaż - %s %02d:00-%02d:00", abbr, h, h+1)
			slots = append(slots, Slot{
				ScheduleID:  schedule.ID,
				SlotType:    SlotTypeMontage,
				StartTime:   slotStart,
				EndTime:     slotEnd,
				CreditHours: 1,
				Capacity:    2,
				Label:       &label,
			})
		}
	}

	// Festival: 1-hour blocks from festival_start to festival_end truncated to full hour.
	festEnd := schedule.FestivalEnd.Truncate(time.Hour)
	cur := schedule.FestivalStart.Truncate(time.Hour).In(warsawLocation)
	for cur.Before(festEnd) {
		slotEnd := cur.Add(time.Hour)
		label := fmt.Sprintf("Festiwal %s", cur.In(warsawLocation).Format("02.01 15:04"))
		slots = append(slots, Slot{
			ScheduleID:  schedule.ID,
			SlotType:    SlotTypeFestival,
			StartTime:   cur,
			EndTime:     slotEnd,
			CreditHours: 1,
			Capacity:    2,
			Label:       &label,
		})
		cur = slotEnd
	}

	// Demontage: event_end day, hourly within window.
	ed := schedule.EventEnd.In(warsawLocation)
	edDay := time.Date(ed.Year(), ed.Month(), ed.Day(), 0, 0, 0, 0, warsawLocation)
	startH, endH := windowFor(edDay)
	deAbbr := dayAbbr[edDay.Weekday()]
	for h := startH; h < endH; h++ {
		slotStart := time.Date(edDay.Year(), edDay.Month(), edDay.Day(), h, 0, 0, 0, warsawLocation)
		slotEnd := slotStart.Add(time.Hour)
		label := fmt.Sprintf("Demontaż - %s %02d:00-%02d:00", deAbbr, h, h+1)
		slots = append(slots, Slot{
			ScheduleID:  schedule.ID,
			SlotType:    SlotTypeDemontage,
			StartTime:   slotStart,
			EndTime:     slotEnd,
			CreditHours: 1,
			Capacity:    2,
			Label:       &label,
		})
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

	dayWindows, err := s.repo.GetDayWindows(id)
	if err != nil {
		return nil, err
	}

	return &ScheduleDetail{
		Schedule:   *schedule,
		Slots:      slotsWithVols,
		Volunteers: volsWithSlots,
		Validation: validation,
		DayWindows: dayWindows,
	}, nil
}

func (s *Service) DeleteSchedule(id int) (bool, error) {
	schedule, err := s.repo.GetSchedule(id)
	if err != nil {
		return false, err
	}
	if schedule == nil {
		return false, nil
	}
	if time.Now().Before(schedule.EventEnd) {
		return false, ErrEventNotEnded
	}
	return s.repo.DeleteSchedule(id)
}

// SaveDraft performs a bulk save of the entire schedule state in a single transaction.
func (s *Service) SaveDraft(req SaveDraftRequest) (*SaveDraftResponse, error) {
	schedule, err := s.getActive()
	if err != nil {
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

func (s *Service) UpsertDayWindow(req UpsertDayWindowRequest) (*DayWindow, error) {
	schedule, err := s.getActive()
	if err != nil {
		return nil, err
	}
	return s.repo.UpsertDayWindow(schedule.ID, req)
}

func (s *Service) DeleteDayWindow(date string) (bool, error) {
	schedule, err := s.getActive()
	if err != nil {
		return false, err
	}
	return s.repo.DeleteDayWindow(schedule.ID, date)
}

type VersionConflictError struct {
	ServerVersion int
	YourVersion   int
}

func (e *VersionConflictError) Error() string {
	return "version_conflict"
}
