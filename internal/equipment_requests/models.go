package equipment_requests

import "time"

// QuestVolunteer is a transfer participant shown on the quest.
type QuestVolunteer struct {
	ID       int     `json:"id"`
	Username string  `json:"username"`
	Fullname *string `json:"fullname"`
}

// QuestTransfer is a transfer linked to a quest, with its current status.
type QuestTransfer struct {
	TransferID int       `json:"transfer_id" db:"transfer_id"`
	Status     string    `json:"status"      db:"transfer_status"`
	CreatedAt  time.Time `json:"created_at"  db:"created_at"`
}

// Quest represents an aggregated equipment release request
type Quest struct {
	ID                 string          `json:"id"` // quest-abc123 (used for all operations)
	QuestKey           string          `json:"-"`  // MD5 hash for deduplication, not exposed in API
	Destination        Destination     `json:"destination"`
	Recipient          string          `json:"recipient"`
	DeliveryDate       string          `json:"delivery_date"`
	PickupTime         string          `json:"pickup_time,omitempty"`
	BudgetOwner        string          `json:"budget_owner"`
	Items              []QuestItem     `json:"items"`
	Status             string          `json:"status"`
	Transfers          []QuestTransfer  `json:"transfers"`           // All linked transfers (empty when none)
	LocationID         *int            `json:"location_id,omitempty"`
	LocationName       *string         `json:"location_name,omitempty"`
	LocationResolved   bool            `json:"location_resolved"`
	AssignedVolunteers []QuestVolunteer `json:"assigned_volunteers"` // Aggregated across all transfers
	SourceRows         []int           `json:"source_rows"`
	LastSynced         time.Time       `json:"last_synced"`
}

// HasActiveTransfer returns true when the quest has at least one non-completed, non-cancelled transfer.
func (q *Quest) HasActiveTransfer() bool {
	for _, t := range q.Transfers {
		if t.Status != "completed" && t.Status != "cancelled" {
			return true
		}
	}
	return false
}

// LocationMapping represents manual pavilion+location_name → location_id override
type LocationMapping struct {
	ID           int       `json:"id" db:"id"`
	Pavilion     string    `json:"pavilion" db:"pavilion"`
	LocationName string    `json:"location_name" db:"location_name"`
	LocationID   int       `json:"location_id" db:"location_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UsageCount   int       `json:"usage_count" db:"usage_count"`
}

// QuestItem represents a single item in a quest.
// Quantity is a pointer because the source sheet may leave it blank — a nil Quantity
// means "not specified yet" (serialised as JSON null) and blocks auto-dispatch until set.
type QuestItem struct {
	Name                    string  `json:"name"`
	Quantity                *int    `json:"quantity"`
	CategoryID              *int    `json:"category_id,omitempty"`
	CategoryMatch           string  `json:"category_match"`                      // exact, fuzzy, manual, none
	CategoryMatchConfidence float64 `json:"category_match_confidence,omitempty"` // 0.00-1.00 for fuzzy matches
	BudgetOwner             string  `json:"budget_owner,omitempty"`              // Per-item budget owner (can differ from quest)
	Notes                   string  `json:"notes,omitempty"`
}

// Destination represents where items should be delivered
type Destination struct {
	Pavilion string `json:"pavilion"`
	Location string `json:"location"`
}

// SheetRow represents a single row from the spreadsheet
type SheetRow struct {
	RowNumber     int
	Item          string
	Quantity      *int // nil when the sheet leaves the quantity cell blank
	Pavilion      string
	Location      string
	Status        string
	PickupTime    string
	DeliveryDate  string
	BudgetOwner   string
	Recipient     string
	Notes         string
	CategoryID    *int   // Filled after matching
	CategoryMatch string // exact, fuzzy, none
}

// CategoryMatch represents the result of matching an item to a category
type CategoryMatch struct {
	CategoryID *int
	MatchType  string // exact, fuzzy, none
	Confidence float64
}

// Status constants
const (
	StatusOrdered   = "Zamówione"
	StatusDelivered = "Dostarczone"
	StatusSent      = "Wysłane"
	StatusReported  = "Zgłoszone"
)

// CreateTransferFromQuestRequest is the request body for creating a transfer from a quest
type CreateTransferFromQuestRequest struct {
	FromLocationID int                 `json:"from_location_id" binding:"required"`
	ToLocationID   *int                `json:"to_location_id"`  // Optional — resolved from quest destination if not provided
	StockItems     []StockItemOverride `json:"stock_items"`     // Optional — auto-resolved from quest items if not provided
	Assets         []AssetOverride     `json:"assets"`          // Optional — serialized assets to include
	Users          []UserOverride      `json:"users,omitempty"` // Optional — users assigned to delivery
}

// StockItemOverride allows the caller to specify exact stock items for the transfer
type StockItemOverride struct {
	ID       int `json:"id" binding:"required"`
	Quantity int `json:"quantity" binding:"required,gte=1"`
}

// AssetOverride allows the caller to specify exact assets for the transfer
type AssetOverride struct {
	ID int `json:"id" binding:"required"`
}

// UserOverride allows the caller to assign users to the transfer
type UserOverride struct {
	ID int `json:"id" binding:"required"`
}

// TransferPreview is returned by the preview endpoint to show what a transfer would look like
type TransferPreview struct {
	FromLocationID  int                 `json:"from_location_id"`
	ToLocationID    *int                `json:"to_location_id,omitempty"`
	ToLocationName  string              `json:"to_location_name,omitempty"`
	ResolvedItems   []ResolvedStockItem `json:"resolved_items"`
	UnresolvedItems []UnresolvedItem    `json:"unresolved_items"`
}

// ResolvedStockItem is a quest item successfully matched to stock at the source location
type ResolvedStockItem struct {
	StockID      int    `json:"stock_id"`
	CategoryID   int    `json:"category_id"`
	CategoryName string `json:"category_name,omitempty"`
	ItemName     string `json:"item_name"`
	Quantity     int    `json:"quantity"`
	Available    int    `json:"available"`
}

// UnresolvedItem is a quest item that could not be matched to stock (or has no quantity)
type UnresolvedItem struct {
	ItemName   string `json:"item_name"`
	Quantity   *int   `json:"quantity"`
	CategoryID *int   `json:"category_id,omitempty"`
	Reason     string `json:"reason"`
}

// QuestEvent is broadcast over SSE. Discriminate on Type field.
//
//	"sync_completed"  — after a Google Sheets sync (Stats populated)
//	"stocks_changed"  — after a stock create/update/delete (LocationID + Action populated)
type QuestEvent struct {
	Type       string     `json:"type"`
	Stats      *SyncStats `json:"stats,omitempty"`
	LocationID int        `json:"location_id,omitempty"`
	Action     string     `json:"action,omitempty"` // "created" | "updated" | "deleted"
}

// SyncStatus describes the current state of the auto-sync scheduler
type SyncStatus struct {
	Enabled   bool       `json:"enabled"`
	Interval  string     `json:"interval,omitempty"`
	LastSync  *time.Time `json:"last_sync,omitempty"`
	NextSync  *time.Time `json:"next_sync,omitempty"`
	LastError string     `json:"last_error,omitempty"`
}
