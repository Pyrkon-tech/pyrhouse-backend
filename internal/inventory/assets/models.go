package assets

import (
	"time"
)

// Asset reprezentuje zasób w systemie
type Asset struct {
	ID         string            `json:"id" db:"id"`
	ItemID     string            `json:"item_id" db:"item_id"`
	Serial     *string           `json:"serial,omitempty" db:"serial"`
	Status     string            `json:"status" db:"status"`
	LocationID *string           `json:"location_id,omitempty" db:"location_id"`
	Origin     string            `json:"origin" db:"origin"`
	CreatedAt  time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at" db:"updated_at"`
	Metadata   map[string]string `json:"metadata,omitempty" db:"metadata"`
}

// AssetRequest reprezentuje żądanie utworzenia/aktualizacji zasobu
type AssetRequest struct {
	ItemID     string            `json:"item_id" binding:"required"`
	Serial     *string           `json:"serial,omitempty"`
	Status     string            `json:"status" binding:"required"`
	LocationID *string           `json:"location_id,omitempty"`
	Origin     string            `json:"origin" binding:"required"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// AssetResponse reprezentuje odpowiedź z danymi zasobu
type AssetResponse struct {
	ID           string            `json:"id"`
	ItemID       string            `json:"item_id"`
	ItemName     string            `json:"item_name,omitempty"`
	Serial       *string           `json:"serial,omitempty"`
	Status       string            `json:"status"`
	LocationID   *string           `json:"location_id,omitempty"`
	LocationName *string           `json:"location_name,omitempty"`
	Origin       string            `json:"origin"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Auditable interface dla zasobów podlegających audytowi
type Auditable interface {
	GetID() string
	GetType() string
	GetAction() string
	GetDetails() map[string]interface{}
}
