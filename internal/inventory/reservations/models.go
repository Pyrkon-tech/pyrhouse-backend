package reservations

import "time"

type PyrCodeReservation struct {
	ID         int        `json:"id"                   db:"id"`
	PyrCode    string     `json:"pyr_code"             db:"pyr_code"`
	CategoryID int        `json:"category_id"          db:"category_id"`
	ReservedAt time.Time  `json:"reserved_at"          db:"reserved_at"`
	ClaimedAt  *time.Time `json:"claimed_at,omitempty" db:"claimed_at"`
	ItemID     *int       `json:"item_id,omitempty"    db:"item_id"`
}

// ReserveRequest — only what's needed to generate and lock pyr_codes.
type ReserveRequest struct {
	CategoryID int `json:"category_id" binding:"required"`
	Quantity   int `json:"quantity"    binding:"required,min=1,max=200"`
}

// ClaimItem — one pyr_code + optional serial (nil = without serial, can be patched later).
type ClaimItem struct {
	PyrCode string  `json:"pyr_code" binding:"required"`
	Serial  *string `json:"serial"`
}

// ClaimRequest — asset properties that apply to all claimed items in this batch.
type ClaimRequest struct {
	Origin     string      `json:"origin"      binding:"required"`
	LocationID int         `json:"location_id"` // optional; defaults to 1
	Items      []ClaimItem `json:"items"       binding:"required,min=1"`
}

type DeleteRequest struct {
	PyrCodes []string `json:"pyr_codes"`
	IDs      []int    `json:"ids"`
}

type ClaimError struct {
	PyrCode string `json:"pyr_code"`
	Reason  string `json:"reason"`
}
