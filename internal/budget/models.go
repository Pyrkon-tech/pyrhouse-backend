package budget

import "time"

// PriceListItem is a single supplier price for one item.
type PriceListItem struct {
	ID        int       `json:"id" db:"id"`
	ItemName  string    `json:"item_name" db:"item_name"`
	Supplier  string    `json:"supplier" db:"supplier"`
	UnitPrice float64   `json:"unit_price" db:"unit_price"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// UpsertPriceRequest creates or updates a single (item_name, supplier) price entry.
type UpsertPriceRequest struct {
	ItemName  string  `json:"item_name" binding:"required"`
	Supplier  string  `json:"supplier" binding:"required"`
	UnitPrice float64 `json:"unit_price" binding:"required,gte=0"`
}

// DeletePriceRequest identifies a specific (item_name, supplier) entry to delete.
type DeletePriceRequest struct {
	ItemName string `form:"item_name" binding:"required"`
	Supplier string `form:"supplier" binding:"required"`
}

// SupplierPrice is one supplier's price contribution for a budget line item.
type SupplierPrice struct {
	Supplier  string  `json:"supplier"`
	UnitPrice float64 `json:"unit_price"`
	Total     float64 `json:"total"`
}

// BudgetItem is one aggregated line in the budget summary.
type BudgetItem struct {
	ItemName string          `json:"item_name"`
	Quantity int             `json:"quantity"`
	Prices   []SupplierPrice `json:"prices"` // one entry per supplier that has a price
}

// SupplierTotal is the grand total for one supplier across all items.
type SupplierTotal struct {
	Supplier string  `json:"supplier"`
	Total    float64 `json:"total"`
}

// BudgetSummary is the full response for GET /budget.
type BudgetSummary struct {
	TotalPositions int             `json:"total_positions"`
	TotalQuantity  int             `json:"total_quantity"`
	SupplierTotals []SupplierTotal `json:"supplier_totals"` // sorted by supplier name
	UnpricedCount  int             `json:"unpriced_count"`
	Items          []BudgetItem    `json:"items"`
}

// Filter controls which quests are included in the budget summary.
type Filter struct {
	BudgetOwner string // empty = all
}
