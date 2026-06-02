package scheduling

import (
	"context"
	"fmt"
	"time"
	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
)

type Repository struct {
	repo *repository.Repository
}

func NewRepository(repo *repository.Repository) *Repository {
	return &Repository{repo: repo}
}

var scheduleReturning = []interface{}{
	"id", "name", "event_start", "event_end", "festival_start", "festival_end",
	"status", "version", "created_at",
}

func (r *Repository) CreateSchedule(req CreateScheduleRequest) (*Schedule, error) {
	var schedule Schedule
	query := r.repo.GoquDBWrapper.Insert("schedules").Rows(goqu.Record{
		"name":           req.Name,
		"event_start":    req.EventStart,
		"event_end":      req.EventEnd,
		"festival_start": req.FestivalStart,
		"festival_end":   req.FestivalEnd,
		"status":         "active",
		"version":        1,
	}).Returning(scheduleReturning...)

	if _, err := query.Executor().ScanStruct(&schedule); err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}
	return &schedule, nil
}

func (r *Repository) GetActiveSchedule() (*Schedule, error) {
	var schedule Schedule
	query := r.repo.GoquDBWrapper.From("schedules").
		Where(goqu.Ex{"status": "active"}).
		Order(goqu.C("created_at").Desc()).
		Limit(1)
	found, err := query.Executor().ScanStruct(&schedule)
	if err != nil {
		return nil, fmt.Errorf("failed to get active schedule: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &schedule, nil
}

func (r *Repository) GetSchedule(id int) (*Schedule, error) {
	var schedule Schedule
	query := r.repo.GoquDBWrapper.From("schedules").Where(goqu.Ex{"id": id})
	found, err := query.Executor().ScanStruct(&schedule)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &schedule, nil
}

func (r *Repository) DeleteSchedule(id int) (bool, error) {
	res, err := r.repo.GoquDBWrapper.Delete("schedules").
		Where(goqu.Ex{"id": id}).
		Executor().Exec()
	if err != nil {
		return false, fmt.Errorf("failed to delete schedule: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *Repository) ArchiveAllSchedules() error {
	_, err := r.repo.GoquDBWrapper.Update("schedules").
		Set(goqu.Record{"status": "archived"}).
		Where(goqu.Ex{"status": "active"}).Executor().Exec()
	return err
}

func (r *Repository) ArchiveAllSchedulesTx(tx *goqu.TxDatabase) error {
	_, err := tx.Update("schedules").
		Set(goqu.Record{"status": "archived"}).
		Where(goqu.Ex{"status": "active"}).Executor().Exec()
	return err
}

func (r *Repository) CreateScheduleTx(tx *goqu.TxDatabase, req CreateScheduleRequest) (*Schedule, error) {
	var schedule Schedule
	query := tx.Insert("schedules").Rows(goqu.Record{
		"name":           req.Name,
		"event_start":    req.EventStart,
		"event_end":      req.EventEnd,
		"festival_start": req.FestivalStart,
		"festival_end":   req.FestivalEnd,
		"status":         "active",
		"version":        1,
	}).Returning(scheduleReturning...)
	if _, err := query.Executor().ScanStruct(&schedule); err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}
	return &schedule, nil
}

func (r *Repository) InsertSlotsReturningTx(tx *goqu.TxDatabase, slots []Slot) error {
	if len(slots) == 0 {
		return nil
	}
	rows := make([]goqu.Record, len(slots))
	for i, s := range slots {
		rows[i] = goqu.Record{
			"schedule_id":  s.ScheduleID,
			"slot_type":    s.SlotType,
			"start_time":   s.StartTime,
			"end_time":     s.EndTime,
			"credit_hours": s.CreditHours,
			"capacity":     s.Capacity,
			"label":        s.Label,
		}
	}
	_, err := tx.Insert("schedule_slots").Rows(rows).Executor().Exec()
	return err
}

// UpdateScheduleStatus updates status and bumps version atomically.
// Returns the updated schedule.
func (r *Repository) UpdateScheduleStatus(id int, status string) (*Schedule, error) {
	var s Schedule
	query := r.repo.GoquDBWrapper.Update("schedules").
		Set(goqu.Record{
			"status":  status,
			"version": goqu.L("version + 1"),
		}).
		Where(goqu.Ex{"id": id}).
		Returning(scheduleReturning...)
	found, err := query.Executor().ScanStruct(&s)
	if err != nil {
		return nil, fmt.Errorf("failed to update schedule status: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("schedule not found")
	}
	return &s, nil
}

// IncrementVersion bumps version inside an existing transaction and returns new version.
func (r *Repository) IncrementVersionTx(tx *goqu.TxDatabase, scheduleID int) (int, error) {
	var s Schedule
	query := tx.Update("schedules").
		Set(goqu.Record{"version": goqu.L("version + 1")}).
		Where(goqu.Ex{"id": scheduleID}).
		Returning("version")
	found, err := query.Executor().ScanStruct(&s)
	if err != nil {
		return 0, fmt.Errorf("failed to increment version: %w", err)
	}
	if !found {
		return 0, fmt.Errorf("schedule not found")
	}
	return s.Version, nil
}

// BumpVersionIfMatchTx atomically increments version only if current version matches expected.
// Returns (true, newVersion) on success, (false, 0) on version conflict.
func (r *Repository) BumpVersionIfMatchTx(tx *goqu.TxDatabase, scheduleID, expectedVersion int) (bool, int, error) {
	var s Schedule
	query := tx.Update("schedules").
		Set(goqu.Record{"version": goqu.L("version + 1")}).
		Where(goqu.Ex{"id": scheduleID, "version": expectedVersion}).
		Returning("version")
	found, err := query.Executor().ScanStruct(&s)
	if err != nil {
		return false, 0, fmt.Errorf("failed to bump version: %w", err)
	}
	return found, s.Version, nil
}

func (r *Repository) InsertVolunteers(scheduleID int, volunteers []Volunteer) error {
	rows := make([]goqu.Record, len(volunteers))
	for i, v := range volunteers {
		rows[i] = goqu.Record{
			"schedule_id":      scheduleID,
			"user_id":          v.UserID,
			"nickname":         v.Nickname,
			"city":             v.City,
			"target_hours":     v.TargetHours,
			"available_from":   v.AvailableFrom,
			"available_to":     v.AvailableTo,
			"notes":            v.Notes,
			"discord_confirmed": v.DiscordConfirmed,
		}
	}
	query := r.repo.GoquDBWrapper.Insert("schedule_volunteers").Rows(rows)
	if _, err := query.Executor().Exec(); err != nil {
		return fmt.Errorf("failed to insert volunteers: %w", err)
	}
	return nil
}

func (r *Repository) UpdateVolunteerDiscordConfirmed(scheduleID int, nickname string, discordConfirmed *string) error {
	_, err := r.repo.GoquDBWrapper.Update("schedule_volunteers").
		Set(goqu.Record{"discord_confirmed": discordConfirmed}).
		Where(goqu.Ex{"schedule_id": scheduleID, "nickname": nickname}).
		Executor().Exec()
	return err
}

// FindConfirmedVolunteer returns the nickname of the volunteer whose discord_confirmed matches
// the given Discord username. Returns ("", false, nil) when not found.
func (r *Repository) FindConfirmedVolunteer(discordUsername string) (nickname string, found bool, err error) {
	var v struct {
		Nickname string `db:"nickname"`
	}
	found, err = r.repo.GoquDBWrapper.
		Select("nickname").
		From("schedule_volunteers").
		Where(goqu.Ex{"discord_confirmed": discordUsername}).
		Limit(1).
		Executor().ScanStruct(&v)
	if err != nil {
		return "", false, fmt.Errorf("failed to find confirmed volunteer: %w", err)
	}
	return v.Nickname, found, nil
}

func (r *Repository) GetVolunteers(scheduleID int) ([]Volunteer, error) {
	var volunteers []Volunteer
	query := r.repo.GoquDBWrapper.From("schedule_volunteers").
		Where(goqu.Ex{"schedule_id": scheduleID}).
		Order(goqu.C("nickname").Asc())
	if err := query.Executor().ScanStructs(&volunteers); err != nil {
		return nil, fmt.Errorf("failed to get volunteers: %w", err)
	}
	return volunteers, nil
}

func (r *Repository) GetSlots(scheduleID int) ([]Slot, error) {
	var slots []Slot
	query := r.repo.GoquDBWrapper.From("schedule_slots").
		Where(goqu.Ex{"schedule_id": scheduleID}).
		Order(goqu.C("start_time").Asc())
	if err := query.Executor().ScanStructs(&slots); err != nil {
		return nil, fmt.Errorf("failed to get slots: %w", err)
	}
	return slots, nil
}

func (r *Repository) DeleteSlots(scheduleID int) error {
	_, err := r.repo.GoquDBWrapper.Delete("schedule_slots").
		Where(goqu.Ex{"schedule_id": scheduleID}).Executor().Exec()
	return err
}

func (r *Repository) DeleteAssignments(scheduleID int) error {
	_, err := r.repo.GoquDBWrapper.Delete("schedule_assignments").
		Where(
			goqu.I("slot_id").In(
				r.repo.GoquDBWrapper.From("schedule_slots").
					Select("id").
					Where(goqu.Ex{"schedule_id": scheduleID}),
			),
		).Executor().Exec()
	return err
}

func (r *Repository) InsertAssignments(assignments []Assignment) error {
	if len(assignments) == 0 {
		return nil
	}
	rows := make([]goqu.Record, len(assignments))
	for i, a := range assignments {
		rows[i] = goqu.Record{
			"slot_id":      a.SlotID,
			"volunteer_id": a.VolunteerID,
		}
	}
	query := r.repo.GoquDBWrapper.Insert("schedule_assignments").Rows(rows)
	if _, err := query.Executor().Exec(); err != nil {
		return fmt.Errorf("failed to insert assignments: %w", err)
	}
	return nil
}

func (r *Repository) GetAssignments(scheduleID int) ([]Assignment, error) {
	var assignments []Assignment
	query := r.repo.GoquDBWrapper.
		Select(goqu.I("a.id"), goqu.I("a.slot_id"), goqu.I("a.volunteer_id")).
		From(goqu.T("schedule_assignments").As("a")).
		InnerJoin(
			goqu.T("schedule_slots").As("s"),
			goqu.On(goqu.Ex{"a.slot_id": goqu.I("s.id")}),
		).
		Where(goqu.Ex{"s.schedule_id": scheduleID})
	if err := query.Executor().ScanStructs(&assignments); err != nil {
		return nil, fmt.Errorf("failed to get assignments: %w", err)
	}
	return assignments, nil
}

// GetAssignmentsWithNicknames returns assignments enriched with volunteer nicknames.
type AssignmentRow struct {
	ID          int    `db:"id"`
	SlotID      int    `db:"slot_id"`
	VolunteerID int    `db:"volunteer_id"`
	Nickname    string `db:"nickname"`
}

func (r *Repository) GetAssignmentsWithNicknames(scheduleID int) ([]AssignmentRow, error) {
	var rows []AssignmentRow
	query := r.repo.GoquDBWrapper.
		Select(
			goqu.I("a.id"),
			goqu.I("a.slot_id"),
			goqu.I("a.volunteer_id"),
			goqu.I("v.nickname"),
		).
		From(goqu.T("schedule_assignments").As("a")).
		InnerJoin(
			goqu.T("schedule_slots").As("s"),
			goqu.On(goqu.Ex{"a.slot_id": goqu.I("s.id")}),
		).
		InnerJoin(
			goqu.T("schedule_volunteers").As("v"),
			goqu.On(goqu.Ex{"a.volunteer_id": goqu.I("v.id")}),
		).
		Where(goqu.Ex{"s.schedule_id": scheduleID})
	if err := query.Executor().ScanStructs(&rows); err != nil {
		return nil, fmt.Errorf("failed to get assignments with nicknames: %w", err)
	}
	return rows, nil
}

func (r *Repository) CreateAssignment(slotID, volunteerID int) (*Assignment, error) {
	var a Assignment
	query := r.repo.GoquDBWrapper.Insert("schedule_assignments").Rows(goqu.Record{
		"slot_id":      slotID,
		"volunteer_id": volunteerID,
	}).Returning("id", "slot_id", "volunteer_id")
	found, err := query.Executor().ScanStruct(&a)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("failed to create assignment: no row returned")
	}
	return &a, nil
}

func (r *Repository) CreateAssignmentTx(tx *goqu.TxDatabase, slotID, volunteerID int) (*Assignment, error) {
	var a Assignment
	query := tx.Insert("schedule_assignments").Rows(goqu.Record{
		"slot_id":      slotID,
		"volunteer_id": volunteerID,
	}).Returning("id", "slot_id", "volunteer_id")
	found, err := query.Executor().ScanStruct(&a)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("failed to create assignment: no row returned")
	}
	return &a, nil
}

func (r *Repository) DeleteAssignment(id int) error {
	_, err := r.repo.GoquDBWrapper.Delete("schedule_assignments").
		Where(goqu.Ex{"id": id}).Executor().Exec()
	return err
}

func (r *Repository) DeleteAssignmentTx(tx *goqu.TxDatabase, id int) error {
	_, err := tx.Delete("schedule_assignments").
		Where(goqu.Ex{"id": id}).Executor().Exec()
	return err
}

func (r *Repository) AssignmentExists(id int) (bool, error) {
	count, err := r.repo.GoquDBWrapper.From("schedule_assignments").
		Where(goqu.Ex{"id": id}).Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Repository) GetAssignment(id int) (*Assignment, error) {
	var a Assignment
	query := r.repo.GoquDBWrapper.From("schedule_assignments").Where(goqu.Ex{"id": id})
	found, err := query.Executor().ScanStruct(&a)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &a, nil
}

func (r *Repository) GetAssignmentWithNickname(id int) (*AssignmentRow, error) {
	var row AssignmentRow
	query := r.repo.GoquDBWrapper.
		Select(
			goqu.I("a.id"),
			goqu.I("a.slot_id"),
			goqu.I("a.volunteer_id"),
			goqu.I("v.nickname"),
		).
		From(goqu.T("schedule_assignments").As("a")).
		InnerJoin(
			goqu.T("schedule_volunteers").As("v"),
			goqu.On(goqu.Ex{"a.volunteer_id": goqu.I("v.id")}),
		).
		Where(goqu.Ex{"a.id": id})
	found, err := query.Executor().ScanStruct(&row)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &row, nil
}

func (r *Repository) UpdateVolunteer(id int, updates map[string]interface{}) (*Volunteer, error) {
	var v Volunteer
	query := r.repo.GoquDBWrapper.Update("schedule_volunteers").
		Set(updates).
		Where(goqu.Ex{"id": id}).
		Returning("id", "schedule_id", "user_id", "nickname", "city", "target_hours", "available_from", "available_to", "notes", "assigned_hours", "discord_confirmed")
	found, err := query.Executor().ScanStruct(&v)
	if err != nil {
		return nil, fmt.Errorf("failed to update volunteer: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &v, nil
}

func (r *Repository) GetVolunteerByID(id int) (*Volunteer, error) {
	var v Volunteer
	found, err := r.repo.GoquDBWrapper.From("schedule_volunteers").
		Where(goqu.Ex{"id": id}).
		Executor().ScanStruct(&v)
	if err != nil {
		return nil, fmt.Errorf("failed to get volunteer: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &v, nil
}

func (r *Repository) GetVolunteerByUserID(scheduleID, userID int) (*Volunteer, error) {
	var v Volunteer
	found, err := r.repo.GoquDBWrapper.From("schedule_volunteers").
		Where(goqu.Ex{"schedule_id": scheduleID, "user_id": userID}).
		Executor().ScanStruct(&v)
	if err != nil {
		return nil, fmt.Errorf("failed to get volunteer by user: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &v, nil
}

type VolunteerSlot struct {
	AssignmentID int       `json:"assignment_id" db:"assignment_id"`
	SlotID       int       `json:"slot_id" db:"slot_id"`
	SlotType     string    `json:"slot_type" db:"slot_type"`
	StartTime    time.Time `json:"start_time" db:"start_time"`
	EndTime      time.Time `json:"end_time" db:"end_time"`
	CreditHours  float64   `json:"credit_hours" db:"credit_hours"`
	Label        *string   `json:"label" db:"label"`
}

func (r *Repository) GetSlotsByVolunteer(volunteerID int) ([]VolunteerSlot, error) {
	var slots []VolunteerSlot
	query := r.repo.GoquDBWrapper.
		Select(
			goqu.I("a.id").As("assignment_id"),
			goqu.I("s.id").As("slot_id"),
			goqu.I("s.slot_type"),
			goqu.I("s.start_time"),
			goqu.I("s.end_time"),
			goqu.I("s.credit_hours"),
			goqu.I("s.label"),
		).
		From(goqu.T("schedule_assignments").As("a")).
		InnerJoin(
			goqu.T("schedule_slots").As("s"),
			goqu.On(goqu.Ex{"a.slot_id": goqu.I("s.id")}),
		).
		Where(goqu.Ex{"a.volunteer_id": volunteerID}).
		Order(goqu.I("s.start_time").Asc())
	if err := query.Executor().ScanStructs(&slots); err != nil {
		return nil, fmt.Errorf("failed to get volunteer slots: %w", err)
	}
	return slots, nil
}

func (r *Repository) DeleteVolunteer(id int) (bool, error) {
	res, err := r.repo.GoquDBWrapper.Delete("schedule_volunteers").
		Where(goqu.Ex{"id": id}).
		Executor().Exec()
	if err != nil {
		return false, fmt.Errorf("failed to delete volunteer: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *Repository) UpdateVolunteerHours(volunteerID int, hours float64) error {
	_, err := r.repo.GoquDBWrapper.Update("schedule_volunteers").
		Set(goqu.Record{"assigned_hours": hours}).
		Where(goqu.Ex{"id": volunteerID}).Executor().Exec()
	return err
}

// Slot CRUD

var slotReturning = []interface{}{"id", "schedule_id", "slot_type", "start_time", "end_time", "credit_hours", "capacity", "label"}

func (r *Repository) CreateSlot(slot Slot) (*Slot, error) {
	var s Slot
	query := r.repo.GoquDBWrapper.Insert("schedule_slots").Rows(goqu.Record{
		"schedule_id":  slot.ScheduleID,
		"slot_type":    slot.SlotType,
		"start_time":   slot.StartTime,
		"end_time":     slot.EndTime,
		"credit_hours": slot.CreditHours,
		"capacity":     slot.Capacity,
		"label":        slot.Label,
	}).Returning(slotReturning...)
	found, err := query.Executor().ScanStruct(&s)
	if err != nil {
		return nil, fmt.Errorf("failed to create slot: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("failed to create slot: no row returned")
	}
	return &s, nil
}

func (r *Repository) GetSlot(id int) (*Slot, error) {
	var s Slot
	found, err := r.repo.GoquDBWrapper.From("schedule_slots").
		Where(goqu.Ex{"id": id}).Executor().ScanStruct(&s)
	if err != nil {
		return nil, fmt.Errorf("failed to get slot: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &s, nil
}

func (r *Repository) UpdateSlotByID(id int, updates map[string]interface{}) (*Slot, error) {
	var s Slot
	query := r.repo.GoquDBWrapper.Update("schedule_slots").
		Set(updates).
		Where(goqu.Ex{"id": id}).
		Returning(slotReturning...)
	found, err := query.Executor().ScanStruct(&s)
	if err != nil {
		return nil, fmt.Errorf("failed to update slot: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &s, nil
}

func (r *Repository) DeleteSlotByID(id int) error {
	_, err := r.repo.GoquDBWrapper.Delete("schedule_slots").
		Where(goqu.Ex{"id": id}).Executor().Exec()
	return err
}

// Transaction-aware methods for bulk draft save

func (r *Repository) UpdateSlotTx(tx *goqu.TxDatabase, id int, record goqu.Record) error {
	_, err := tx.Update("schedule_slots").
		Set(record).
		Where(goqu.Ex{"id": id}).Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to update slot %d: %w", id, err)
	}
	return nil
}

func (r *Repository) InsertSlotTx(tx *goqu.TxDatabase, slot Slot) (*Slot, error) {
	var s Slot
	query := tx.Insert("schedule_slots").Rows(goqu.Record{
		"schedule_id":  slot.ScheduleID,
		"slot_type":    slot.SlotType,
		"start_time":   slot.StartTime,
		"end_time":     slot.EndTime,
		"credit_hours": slot.CreditHours,
		"capacity":     slot.Capacity,
		"label":        slot.Label,
	}).Returning(slotReturning...)
	found, err := query.Executor().ScanStruct(&s)
	if err != nil {
		return nil, fmt.Errorf("failed to insert slot: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("failed to insert slot: no row returned")
	}
	return &s, nil
}

func (r *Repository) DeleteSlotsByIDsTx(tx *goqu.TxDatabase, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	vals := make([]interface{}, len(ids))
	for i, id := range ids {
		vals[i] = id
	}
	_, err := tx.Delete("schedule_slots").
		Where(goqu.C("id").In(vals...)).Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to delete slots: %w", err)
	}
	return nil
}

func (r *Repository) DeleteAssignmentsByScheduleTx(tx *goqu.TxDatabase, scheduleID int) error {
	_, err := tx.Delete("schedule_assignments").
		Where(
			goqu.I("slot_id").In(
				tx.From("schedule_slots").
					Select("id").
					Where(goqu.Ex{"schedule_id": scheduleID}),
			),
		).Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to delete assignments: %w", err)
	}
	return nil
}

func (r *Repository) InsertAssignmentsTx(tx *goqu.TxDatabase, assignments []Assignment) error {
	if len(assignments) == 0 {
		return nil
	}
	rows := make([]goqu.Record, len(assignments))
	for i, a := range assignments {
		rows[i] = goqu.Record{
			"slot_id":      a.SlotID,
			"volunteer_id": a.VolunteerID,
		}
	}
	_, err := tx.Insert("schedule_assignments").Rows(rows).Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to insert assignments: %w", err)
	}
	return nil
}

func (r *Repository) RecalcVolunteerHoursTx(tx *goqu.TxDatabase, scheduleID int) error {
	sql := `
		UPDATE schedule_volunteers sv
		SET assigned_hours = COALESCE(sub.total, 0)
		FROM (
			SELECT a.volunteer_id, SUM(s.credit_hours) AS total
			FROM schedule_assignments a
			JOIN schedule_slots s ON s.id = a.slot_id
			WHERE s.schedule_id = $1
			GROUP BY a.volunteer_id
		) sub
		WHERE sv.id = sub.volunteer_id AND sv.schedule_id = $1
	`
	if _, err := tx.Exec(sql, scheduleID); err != nil {
		return fmt.Errorf("failed to recalc volunteer hours: %w", err)
	}

	sqlZero := `
		UPDATE schedule_volunteers
		SET assigned_hours = 0
		WHERE schedule_id = $1
		AND id NOT IN (
			SELECT DISTINCT a.volunteer_id
			FROM schedule_assignments a
			JOIN schedule_slots s ON s.id = a.slot_id
			WHERE s.schedule_id = $1
		)
	`
	if _, err := tx.Exec(sqlZero, scheduleID); err != nil {
		return fmt.Errorf("failed to zero volunteer hours: %w", err)
	}

	return nil
}

// DB returns the goqu database wrapper for transactions.
func (r *Repository) DB() *goqu.Database {
	return r.repo.GoquDBWrapper
}

// Day window CRUD

func (r *Repository) GetDayWindows(scheduleID int) ([]DayWindow, error) {
	var windows []DayWindow
	query := r.repo.GoquDBWrapper.From("schedule_day_windows").
		Where(goqu.Ex{"schedule_id": scheduleID}).
		Order(goqu.C("date").Asc())
	if err := query.Executor().ScanStructs(&windows); err != nil {
		return nil, fmt.Errorf("failed to get day windows: %w", err)
	}
	if windows == nil {
		windows = []DayWindow{}
	}
	return windows, nil
}

// GetDayWindowsAsMap returns a map of date string → [startHour, endHour] for fast lookup.
func (r *Repository) GetDayWindowsAsMap(scheduleID int) (map[string][2]int, error) {
	windows, err := r.GetDayWindows(scheduleID)
	if err != nil {
		return nil, err
	}
	m := make(map[string][2]int, len(windows))
	for _, w := range windows {
		var sh, eh int
		fmt.Sscanf(w.WindowStart, "%d", &sh)
		fmt.Sscanf(w.WindowEnd, "%d", &eh)
		m[w.Date] = [2]int{sh, eh}
	}
	return m, nil
}

func (r *Repository) UpsertDayWindow(scheduleID int, req UpsertDayWindowRequest) (*DayWindow, error) {
	const sql = `
		INSERT INTO schedule_day_windows (schedule_id, date, window_start, window_end)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (schedule_id, date) DO UPDATE
		  SET window_start = EXCLUDED.window_start,
		      window_end   = EXCLUDED.window_end
		RETURNING id, schedule_id, date::text, window_start::text, window_end::text
	`
	row := r.repo.DB.QueryRow(sql, scheduleID, req.Date, req.WindowStart, req.WindowEnd)
	var w DayWindow
	if err := row.Scan(&w.ID, &w.ScheduleID, &w.Date, &w.WindowStart, &w.WindowEnd); err != nil {
		return nil, fmt.Errorf("failed to upsert day window: %w", err)
	}
	return &w, nil
}

func (r *Repository) DeleteDayWindow(scheduleID int, date string) (bool, error) {
	res, err := r.repo.GoquDBWrapper.Delete("schedule_day_windows").
		Where(goqu.Ex{"schedule_id": scheduleID, "date": date}).
		Executor().Exec()
	if err != nil {
		return false, fmt.Errorf("failed to delete day window: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *Repository) DeleteNonFestivalSlotsTx(tx *goqu.TxDatabase, scheduleID int) error {
	_, err := tx.Delete("schedule_slots").
		Where(goqu.Ex{
			"schedule_id": scheduleID,
			"slot_type":   goqu.Op{"neq": SlotTypeFestival},
		}).Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to delete non-festival slots: %w", err)
	}
	return nil
}

// GetOnDutyVolunteers returns volunteers assigned to slots active at time at,
// enriched with real-time dispatch status derived from in-progress quests.
func (r *Repository) GetOnDutyVolunteers(ctx context.Context, at time.Time) ([]OnDutyEntry, error) {
	const query = `
		SELECT
			v.id          AS volunteer_id,
			v.nickname,
			s.id          AS slot_id,
			s.label       AS slot_label,
			s.end_time    AS slot_end,
			CASE WHEN aq.user_id IS NOT NULL THEN 'on_mission' ELSE 'available' END AS status,
			CASE
				WHEN aq.user_id IS NOT NULL
				THEN aq.destination_pavilion
				     || COALESCE(' - ' || NULLIF(TRIM(aq.destination_location), ''), '')
				ELSE NULL
			END AS current_mission,
			v.user_id,
			u.username,
			u.fullname,
			u.avatar_url,
			u.discord_username
		FROM schedules sc
		JOIN schedule_slots s        ON s.schedule_id = sc.id
		JOIN schedule_assignments a  ON a.slot_id = s.id
		JOIN schedule_volunteers v   ON v.id = a.volunteer_id
		LEFT JOIN users u            ON u.id = v.user_id
		LEFT JOIN (
			SELECT DISTINCT ON (tu.user_id)
				tu.user_id,
				q.destination_pavilion,
				q.destination_location
			FROM transfer_users tu
			JOIN transfers t                ON tu.transfer_id = t.id
			JOIN quest_transfers qt         ON qt.transfer_id = t.id
			JOIN equipment_request_quests q ON qt.quest_id    = q.quest_id
			WHERE q.status = 'in_progress'
			ORDER BY tu.user_id, t.id DESC
		) aq ON u.id = aq.user_id
		WHERE sc.status = 'active'
		  AND s.start_time <= $1
		  AND s.end_time   >  $1
		ORDER BY v.nickname ASC
	`
	rows, err := r.repo.DB.QueryContext(ctx, query, at)
	if err != nil {
		return nil, fmt.Errorf("on-duty query failed: %w", err)
	}
	defer rows.Close()

	var entries []OnDutyEntry
	for rows.Next() {
		var e OnDutyEntry
		var (
			username        *string
			fullname        *string
			avatarURL       *string
			discordUsername *string
		)
		if err := rows.Scan(
			&e.VolunteerID,
			&e.Nickname,
			&e.SlotID,
			&e.SlotLabel,
			&e.SlotEnd,
			&e.Status,
			&e.CurrentMission,
			&e.UserID,
			&username,
			&fullname,
			&avatarURL,
			&discordUsername,
		); err != nil {
			return nil, fmt.Errorf("on-duty scan failed: %w", err)
		}
		if e.UserID != nil && username != nil {
			e.User = &UserInfo{
				ID:              *e.UserID,
				Username:        *username,
				Fullname:        fullname,
				AvatarURL:       avatarURL,
				DiscordUsername: discordUsername,
			}
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("on-duty rows error: %w", err)
	}
	if entries == nil {
		entries = []OnDutyEntry{}
	}
	return entries, nil
}
