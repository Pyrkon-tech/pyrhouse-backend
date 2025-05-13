package service_desk

import (
	"time"
)

type RequestType struct {
	Type        string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

type Request struct {
	ID          int       `json:"id,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	CreatedBy   string    `json:"created_by"`
	AssignedTo  *int      `json:"assigned_to,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	Priority    string    `json:"priority"`
	Location    *string   `json:"location,omitempty"`
	CreatedByID *int      `json:"created_by_id,omitempty"`
}

type RequestComment struct {
	ID        int       `json:"id" db:"id"`
	RequestID int       `json:"request_id" db:"request_id"`
	Content   string    `json:"content" db:"comment"`
	UserID    int       `json:"created_by" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Comment struct {
	ID        int       `json:"id"`
	RequestID int       `json:"request_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	User      *User     `json:"user"`
}

type FlatComment struct {
	ID        int       `db:"id"`
	RequestID int       `db:"request_id"`
	Content   string    `db:"comment"`
	CreatedAt time.Time `db:"created_at"`
	UserID    int       `db:"user_id"`
	Username  string    `db:"comment_user_username"`
	Fullname  string    `db:"comment_user_fullname"`
}

type RequestResponse struct {
	ID             int       `json:"id,omitempty"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Status         string    `json:"status"`
	CreatedBy      string    `json:"created_by"`
	Type           string    `json:"type"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
	Priority       string    `json:"priority"`
	Location       *string   `json:"location,omitempty"`
	CreatedByUser  *User     `json:"created_by_user,omitempty"`
	AssignedToUser *User     `json:"assigned_to_user,omitempty"`
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Fullname string `json:"fullname"`
}

type FlatRequestResponse struct {
	ID                 int       `json:"id,omitempty" db:"id"`
	Title              string    `json:"title" db:"title"`
	Type               string    `json:"type" db:"type"`
	Description        string    `json:"description" db:"description"`
	Status             string    `json:"status" db:"status"`
	CreatedBy          string    `json:"created_by" db:"created_by"`
	CreatedAt          time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at,omitempty" db:"updated_at"`
	Priority           string    `json:"priority" db:"priority"`
	Location           *string   `json:"location,omitempty" db:"location"`
	UserID             *int      `json:"user_id" db:"created_by_id"`
	UserUsername       *string   `json:"user_username" db:"request_user_username"`
	UserFullname       *string   `json:"user_fullname" db:"request_user_fullname"`
	AssignedToID       *int      `json:"assigned_to_id,omitempty" db:"assigned_to_id"`
	AssignedToUsername *string   `json:"assigned_to_username" db:"request_assigned_to_username"`
	AssignedToFullname *string   `json:"assigned_to_fullname" db:"request_assigned_to_fullname"`
}

func (fr *FlatRequestResponse) TransformToRequestResponse() *RequestResponse {
	res := RequestResponse{
		ID:          fr.ID,
		Title:       fr.Title,
		Description: fr.Description,
		Type:        fr.Type,
		Status:      fr.Status,
		CreatedBy:   fr.CreatedBy,
		CreatedAt:   fr.CreatedAt,
		UpdatedAt:   fr.UpdatedAt,
		Priority:    fr.Priority,
		Location:    fr.Location,
	}

	if fr.UserID != nil {
		res.CreatedByUser = &User{
			ID:       *fr.UserID,
			Username: *fr.UserUsername,
			Fullname: *fr.UserFullname,
		}
	}

	if fr.AssignedToID != nil {
		res.AssignedToUser = &User{
			ID:       *fr.AssignedToID,
			Username: *fr.AssignedToUsername,
			Fullname: *fr.AssignedToFullname,
		}
	}

	return &res
}

const (
	RequestTypeHardwareIssue    = "hardware_issue"
	RequestTypeReplacement      = "replacement"
	RequestTypeTechnicalProblem = "technical_problem"
	RequestTypeOther            = "other"
)

const (
	StatusNew        = "new"
	StatusInProgress = "in_progress"
	StatusWaiting    = "waiting"
	StatusResolved   = "resolved"
	StatusClosed     = "closed"
)

const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)
