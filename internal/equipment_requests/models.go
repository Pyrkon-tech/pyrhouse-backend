package equipment_requests

import "time"

// Quest represents an aggregated equipment release request
type Quest struct {
	ID           string       `json:"id"` // quest-abc123 (used for all operations)
	QuestKey     string       `json:"-"` // MD5 hash for deduplication, not exposed in API
	Destination  Destination  `json:"destination"`
	Recipient    string       `json:"recipient"`
	DeliveryDate string       `json:"delivery_date"`
	PickupTime   string       `json:"pickup_time,omitempty"`
	BudgetOwner  string       `json:"budget_owner"`
	Items        []QuestItem  `json:"items"`
	Status       string       `json:"status"`
	SourceRows   []int        `json:"source_rows"`
	LastSynced   time.Time    `json:"last_synced"`
}

// QuestItem represents a single item in a quest
type QuestItem struct {
	Name                    string  `json:"name"`
	Quantity                int     `json:"quantity"`
	CategoryID              *int    `json:"category_id,omitempty"`
	CategoryMatch           string  `json:"category_match"` // exact, fuzzy, manual, none
	CategoryMatchConfidence float64 `json:"category_match_confidence,omitempty"` // 0.00-1.00 for fuzzy matches
	BudgetOwner             string  `json:"budget_owner,omitempty"` // Per-item budget owner (can differ from quest)
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
	Quantity      int
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
	MatchType  string  // exact, fuzzy, none
	Confidence float64
}

// Status constants
const (
	StatusOrdered   = "Zamówione"
	StatusDelivered = "Dostarczone"
	StatusSent      = "Wysłane"
	StatusReported  = "Zgłoszone"
)
