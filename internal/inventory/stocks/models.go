package stocks

import (
	"time"
)

// StockItem reprezentuje pozycję w magazynie
type StockItem struct {
	ID         string            `json:"id" db:"id"`
	ItemID     string            `json:"item_id" db:"item_id"`
	Quantity   int               `json:"quantity" db:"quantity"`
	Status     string            `json:"status" db:"status"`
	LocationID *string           `json:"location_id,omitempty" db:"location_id"`
	CreatedAt  time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at" db:"updated_at"`
	Metadata   map[string]string `json:"metadata,omitempty" db:"metadata"`
}

// StockRequest reprezentuje żądanie utworzenia/aktualizacji pozycji magazynowej
type StockRequest struct {
	ItemID     string            `json:"item_id" binding:"required"`
	Quantity   int               `json:"quantity" binding:"required,min=0"`
	Status     string            `json:"status" binding:"required"`
	LocationID *string           `json:"location_id,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// StockResponse reprezentuje odpowiedź z danymi pozycji magazynowej
type StockResponse struct {
	ID           string            `json:"id"`
	ItemID       string            `json:"item_id"`
	ItemName     string            `json:"item_name,omitempty"`
	Quantity     int               `json:"quantity"`
	Status       string            `json:"status"`
	LocationID   *string           `json:"location_id,omitempty"`
	LocationName *string           `json:"location_name,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}
