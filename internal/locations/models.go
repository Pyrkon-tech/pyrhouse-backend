package locations

import (
	"time"
)

// Location reprezentuje lokalizację w systemie
type Location struct {
	ID          string            `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description" db:"description"`
	Pavilion    *string           `json:"pavilion,omitempty" db:"pavilion"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
	Metadata    map[string]string `json:"metadata,omitempty" db:"metadata"`
}

// LocationRequest reprezentuje żądanie utworzenia/aktualizacji lokalizacji
type LocationRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Pavilion    *string           `json:"pavilion,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// LocationResponse reprezentuje odpowiedź z danymi lokalizacji
type LocationResponse struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Pavilion    *string           `json:"pavilion,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
