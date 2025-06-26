package models

import (
	"encoding/json"
	"time"
)

type AuditLog struct {
	ID           int                    `json:"id" db:"id"`
	ResourceID   int                    `json:"resource_id" db:"resource_id"`
	ResourceType string                 `json:"resource_type" db:"resource_type"`
	Action       string                 `json:"action" db:"action"` // Captures what happened (e.g., create, update, delete, in_transfer, delivered).
	DataRaw      string                 `json:"-" db:"data"`        // JSON as string
	Data         map[string]interface{} `json:"data" db:"-"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UserID       *int                   `json:"user_id,omitempty" db:"user_id"`
}

func (a *AuditLog) LoadFromDB() {
	if a.DataRaw != "" {
		_ = json.Unmarshal([]byte(a.DataRaw), &a.Data)
	}
}
