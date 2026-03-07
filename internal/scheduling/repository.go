package scheduling

import (
	"fmt"
	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
)

type Repository struct {
	repo *repository.Repository
}

func NewRepository(repo *repository.Repository) *Repository {
	return &Repository{repo: repo}
}

func (r *Repository) CreateSchedule(req CreateScheduleRequest) (*Schedule, error) {
	var schedule Schedule
	query := r.repo.GoquDBWrapper.Insert("schedules").Rows(goqu.Record{
		"name":           req.Name,
		"event_start":    req.EventStart,
		"event_end":      req.EventEnd,
		"festival_start": req.FestivalStart,
		"festival_end":   req.FestivalEnd,
	}).Returning("id", "name", "event_start", "event_end", "festival_start", "festival_end", "status", "created_at")

	if _, err := query.Executor().ScanStruct(&schedule); err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}
	return &schedule, nil
}

func (r *Repository) GetSchedules() ([]Schedule, error) {
	var schedules []Schedule
	query := r.repo.GoquDBWrapper.From("schedules").Order(goqu.C("created_at").Desc())
	if err := query.Executor().ScanStructs(&schedules); err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}
	return schedules, nil
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

func (r *Repository) InsertVolunteers(scheduleID int, volunteers []Volunteer) error {
	rows := make([]goqu.Record, len(volunteers))
	for i, v := range volunteers {
		rows[i] = goqu.Record{
			"schedule_id":    scheduleID,
			"user_id":        v.UserID,
			"nickname":       v.Nickname,
			"city":           v.City,
			"target_hours":   v.TargetHours,
			"available_from": v.AvailableFrom,
			"available_to":   v.AvailableTo,
			"notes":          v.Notes,
		}
	}
	query := r.repo.GoquDBWrapper.Insert("schedule_volunteers").Rows(rows)
	if _, err := query.Executor().Exec(); err != nil {
		return fmt.Errorf("failed to insert volunteers: %w", err)
	}
	return nil
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

func (r *Repository) InsertSlots(slots []Slot) error {
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
	query := r.repo.GoquDBWrapper.Insert("schedule_slots").Rows(rows)
	if _, err := query.Executor().Exec(); err != nil {
		return fmt.Errorf("failed to insert slots: %w", err)
	}
	return nil
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

func (r *Repository) DeleteAssignment(id int) error {
	_, err := r.repo.GoquDBWrapper.Delete("schedule_assignments").
		Where(goqu.Ex{"id": id}).Executor().Exec()
	return err
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

func (r *Repository) UpdateVolunteerHours(volunteerID int, hours float64) error {
	_, err := r.repo.GoquDBWrapper.Update("schedule_volunteers").
		Set(goqu.Record{"assigned_hours": hours}).
		Where(goqu.Ex{"id": volunteerID}).Executor().Exec()
	return err
}

func (r *Repository) UpdateScheduleStatus(id int, status string) error {
	_, err := r.repo.GoquDBWrapper.Update("schedules").
		Set(goqu.Record{"status": status}).
		Where(goqu.Ex{"id": id}).Executor().Exec()
	return err
}
