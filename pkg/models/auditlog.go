package models

import "time"

type AuditLog struct {
	ID           int       `json:"id"`
	ResourceID   int       `json:"resource_id"`
	ResourceType string    `json:"resource_type"`
	Action       string    `json:"action"` // Captures what happened (e.g., create, update, delete, in_transfer, delivered).
	Data         string    `json:"data"`   // JSON as string
	CreatedAt    time.Time `json:"created_at"`
	UserID       *int      `json:"user_id,omitempty"`
}
