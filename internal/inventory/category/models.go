package category

import (
	"time"
)

// ItemCategory reprezentuje kategorię przedmiotów
type ItemCategory struct {
	ID          string            `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description" db:"description"`
	ParentID    *string           `json:"parent_id,omitempty" db:"parent_id"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
	Metadata    map[string]string `json:"metadata,omitempty" db:"metadata"`
}

// CategoryRequest reprezentuje żądanie utworzenia/aktualizacji kategorii
type CategoryRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	ParentID    *string           `json:"parent_id,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CategoryResponse reprezentuje odpowiedź z danymi kategorii
type CategoryResponse struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	ParentID    *string           `json:"parent_id,omitempty"`
	ParentName  *string           `json:"parent_name,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
