package scheduling

import "time"

// Database models

type Schedule struct {
	ID            int       `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	EventStart    string    `json:"event_start" db:"event_start"`
	EventEnd      string    `json:"event_end" db:"event_end"`
	FestivalStart time.Time `json:"festival_start" db:"festival_start"`
	FestivalEnd   time.Time `json:"festival_end" db:"festival_end"`
	Status        string    `json:"status" db:"status"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

type Volunteer struct {
	ID            int       `json:"id" db:"id"`
	ScheduleID    int       `json:"schedule_id" db:"schedule_id"`
	UserID        *int      `json:"user_id" db:"user_id"`
	Nickname      string    `json:"nickname" db:"nickname"`
	City          *string   `json:"city" db:"city"`
	TargetHours   int       `json:"target_hours" db:"target_hours"`
	AvailableFrom time.Time `json:"available_from" db:"available_from"`
	AvailableTo   time.Time `json:"available_to" db:"available_to"`
	Notes         *string   `json:"notes" db:"notes"`
	AssignedHours float64   `json:"assigned_hours" db:"assigned_hours"`
}

type Slot struct {
	ID          int       `json:"id" db:"id"`
	ScheduleID  int       `json:"schedule_id" db:"schedule_id"`
	SlotType    string    `json:"type" db:"slot_type"`
	StartTime   time.Time `json:"start" db:"start_time"`
	EndTime     time.Time `json:"end" db:"end_time"`
	CreditHours float64   `json:"credit_hours" db:"credit_hours"`
	Capacity    int       `json:"capacity" db:"capacity"`
	Label       *string   `json:"label" db:"label"`
}

type Assignment struct {
	ID          int `json:"id" db:"id"`
	SlotID      int `json:"slot_id" db:"slot_id"`
	VolunteerID int `json:"volunteer_id" db:"volunteer_id"`
}

// API request types

type CreateScheduleRequest struct {
	Name          string `json:"name" binding:"required"`
	EventStart    string `json:"event_start" binding:"required"`
	EventEnd      string `json:"event_end" binding:"required"`
	FestivalStart string `json:"festival_start" binding:"required"`
	FestivalEnd   string `json:"festival_end" binding:"required"`
}

type VolunteerInput struct {
	Nickname      string  `json:"nickname" binding:"required"`
	City          *string `json:"city"`
	Hours         int     `json:"hours" binding:"required"`
	AvailableFrom string  `json:"available_from" binding:"required"`
	AvailableTo   string  `json:"available_to" binding:"required"`
	Notes         *string `json:"notes"`
	UserID        *int    `json:"user_id"`
}

type ImportVolunteersRequest struct {
	Volunteers []VolunteerInput `json:"volunteers" binding:"required,min=1"`
}

type SwapRequest struct {
	AssignmentA int `json:"assignment_a" binding:"required"`
	AssignmentB int `json:"assignment_b" binding:"required"`
}

// API response types

type SlotWithVolunteers struct {
	Slot
	Volunteers []VolunteerBrief `json:"volunteers"`
}

type VolunteerBrief struct {
	ID       int    `json:"id"`
	Nickname string `json:"nickname"`
}

type VolunteerWithSlots struct {
	Volunteer
	SlotIDs []int `json:"slots"`
}

type ScheduleDetail struct {
	Schedule
	Slots      []SlotWithVolunteers `json:"slots"`
	Volunteers []VolunteerWithSlots `json:"volunteers"`
	Validation *ValidationResult    `json:"validation"`
}

type ValidationIssue struct {
	Type      string `json:"type"`
	Volunteer string `json:"volunteer,omitempty"`
	Slot      string `json:"slot,omitempty"`
	Assigned  int    `json:"assigned,omitempty"`
	Target    int    `json:"target,omitempty"`
	Capacity  int    `json:"capacity,omitempty"`
}

type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Issues []ValidationIssue `json:"issues"`
}

// Slot types
const (
	SlotTypeMontage   = "montage"
	SlotTypeFestival  = "festival"
	SlotTypeDemontage = "demontage"
)
