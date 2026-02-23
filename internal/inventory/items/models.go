package items

import (
	"time"
)

// Item reprezentuje przedmiot w systemie
type Item struct {
	ID          string            `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description" db:"description"`
	CategoryID  string            `json:"category_id" db:"category_id"`
	Origin      string            `json:"origin" db:"origin"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
	Metadata    map[string]string `json:"metadata,omitempty" db:"metadata"`
}

// ItemRequest reprezentuje żądanie utworzenia/aktualizacji przedmiotu
type ItemRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	CategoryID  string            `json:"category_id" binding:"required"`
	Origin      string            `json:"origin" binding:"required"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ItemQuery reprezentuje parametry zapytania o przedmioty
type ItemQuery struct {
	Search     string `form:"search"`
	CategoryID string `form:"category_id"`
	Origin     string `form:"origin"`
	Limit      int    `form:"limit,default=10"`
	Offset     int    `form:"offset,default=0"`
	SortBy     string `form:"sort_by,default=created_at"`
	SortOrder  string `form:"sort_order,default=desc"`
}

// ItemResponse reprezentuje odpowiedź z danymi przedmiotu
type ItemResponse struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	CategoryID   string            `json:"category_id"`
	CategoryName string            `json:"category_name,omitempty"`
	Origin       string            `json:"origin"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}
