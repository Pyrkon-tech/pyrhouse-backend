package scheduling

import (
	"errors"
	"time"
)

var ErrVersionConflict = errors.New("version conflict")
var ErrNoActiveSchedule = errors.New("no active schedule")
var ErrSlotNotFound = errors.New("slot not found")
var ErrFestivalSlot = errors.New("festival slot")
var ErrFestivalSlotType = errors.New("cannot change type of festival slot")
var ErrEventNotEnded = errors.New("event not ended")
var ErrAssignmentNotFound = errors.New("assignment not found")
var ErrAssignmentsNotFound = errors.New("one or both assignments not found")
var ErrSheetsUnavailable = errors.New("Google Sheets integration not available")

type GenerateBlockedError struct {
	Volunteers []string
}

func (e *GenerateBlockedError) Error() string {
	return "generate_blocked"
}

// Database models

type Schedule struct {
	ID            int       `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	EventStart    time.Time `json:"event_start" db:"event_start"`
	EventEnd      time.Time `json:"event_end" db:"event_end"`
	FestivalStart time.Time `json:"festival_start" db:"festival_start"`
	FestivalEnd   time.Time `json:"festival_end" db:"festival_end"`
	Status        string    `json:"status" db:"status"`
	Version       int       `json:"version" db:"version"`
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

type ImportFromSheetRequest struct {
	SheetID   string `json:"sheet_id" binding:"required"`
	SheetName string `json:"sheet_name" binding:"required"`
}

type UpdateVolunteerRequest struct {
	Nickname      *string `json:"nickname"`
	City          *string `json:"city"`
	Hours         *int    `json:"hours"`
	AvailableFrom *string `json:"available_from"`
	AvailableTo   *string `json:"available_to"`
	Notes         *string `json:"notes"`
	UserID        *int    `json:"user_id"`
}

type AddAssignmentRequest struct {
	VolunteerID int `json:"volunteer_id" binding:"required"`
	SlotID      int `json:"slot_id" binding:"required"`
}

type MoveAssignmentRequest struct {
	AssignmentID int `json:"assignment_id" binding:"required"`
	ToSlotID     int `json:"to_slot_id" binding:"required"`
}

type SwapRequest struct {
	AssignmentA int `json:"assignment_a" binding:"required"`
	AssignmentB int `json:"assignment_b" binding:"required"`
}

type CreateSlotRequest struct {
	Type     string  `json:"type" binding:"required"`
	Start    string  `json:"start" binding:"required"`
	End      string  `json:"end" binding:"required"`
	Capacity int     `json:"capacity"`
	Label    *string `json:"label"`
}

type UpdateSlotRequest struct {
	Type     *string `json:"type"`
	Start    *string `json:"start"`
	End      *string `json:"end"`
	Capacity *int    `json:"capacity"`
	Label    *string `json:"label"`
}

type DraftSlot struct {
	ID       *int    `json:"id,omitempty"`
	TempID   *string `json:"temp_id,omitempty"`
	Type     string  `json:"type" binding:"required"`
	Start    string  `json:"start" binding:"required"`
	End      string  `json:"end" binding:"required"`
	Capacity int     `json:"capacity"`
	Label    *string `json:"label"`
}

type DraftAssignment struct {
	VolunteerID int     `json:"volunteer_id" binding:"required"`
	SlotID      *int    `json:"slot_id,omitempty"`
	SlotTempID  *string `json:"slot_temp_id,omitempty"`
}

type SaveDraftRequest struct {
	Version     int               `json:"version"`
	Slots       []DraftSlot       `json:"slots"`
	Assignments []DraftAssignment `json:"assignments"`
}

type SaveDraftResponse struct {
	Schedule     ScheduleDetail  `json:"schedule"`
	CreatedSlots []TempIDMapping `json:"created_slots"`
	Validation   *ValidationResult `json:"validation"`
}

type TempIDMapping struct {
	TempID string `json:"temp_id"`
	ID     int    `json:"id"`
}

// API response types

// SlotVolunteer represents a volunteer assigned to a slot.
// ID is the assignment ID (used for DELETE and move/swap operations).
type SlotVolunteer struct {
	ID          int    `json:"id"`           // assignment_id
	VolunteerID int    `json:"volunteer_id"`
	Nickname    string `json:"nickname"`
}

type SlotWithVolunteers struct {
	Slot
	Volunteers []SlotVolunteer `json:"volunteers"`
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

type AssignmentDetail struct {
	ID          int    `json:"id"`
	SlotID      int    `json:"slot_id"`
	VolunteerID int    `json:"volunteer_id"`
	Nickname    string `json:"nickname"`
}

type SwapResponse struct {
	AssignmentA AssignmentDetail `json:"assignment_a"`
	AssignmentB AssignmentDetail `json:"assignment_b"`
}

type MoveResponse struct {
	DeletedAssignmentID int              `json:"deleted_assignment_id"`
	CreatedAssignment   AssignmentDetail `json:"created_assignment"`
}

type ImportResult struct {
	Imported int `json:"imported"`
	Updated  int `json:"updated"`
	Skipped  int `json:"skipped"`
}

type ImportSheetResult struct {
	Imported int      `json:"imported"`
	Updated  int      `json:"updated"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

type ValidationIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Volunteer   string `json:"volunteer,omitempty"`
	VolunteerID *int   `json:"volunteer_id,omitempty"`
	Slot        string `json:"slot,omitempty"`
	SlotID      *int   `json:"slot_id,omitempty"`
	Assigned    int    `json:"assigned,omitempty"`
	Target      int    `json:"target,omitempty"`
	Capacity    int    `json:"capacity,omitempty"`
	Message     string `json:"message"`
}

type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Issues []ValidationIssue `json:"issues"`
}

// OnDutyEntry represents a volunteer currently on duty (assigned to an active slot).
type OnDutyEntry struct {
	VolunteerID    int       `json:"volunteer_id"`
	Nickname       string    `json:"nickname"`
	SlotID         int       `json:"slot_id"`
	SlotLabel      *string   `json:"slot_label"`
	SlotEnd        time.Time `json:"slot_end"`
	Status         string    `json:"status"`
	CurrentMission *string   `json:"current_mission"`
	UserID         *int      `json:"user_id"`
	User           *UserInfo `json:"user"`
}

// UserInfo is the user detail embedded in OnDutyEntry.
type UserInfo struct {
	ID              int     `json:"id"`
	Username        string  `json:"username"`
	Fullname        *string `json:"fullname"`
	AvatarURL       *string `json:"avatar_url"`
	DiscordUsername *string `json:"discord_username"`
}

// Slot types
const (
	SlotTypeMontage   = "montage"
	SlotTypeFestival  = "festival"
	SlotTypeDemontage = "demontage"
)
