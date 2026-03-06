package releases

import "time"

type Release struct {
	ID          int        `json:"id" db:"id"`
	Reference   string     `json:"reference" db:"reference"`
	OriginID    *int       `json:"origin_id" db:"origin_id"`
	OriginLabel *string    `json:"origin_label,omitempty" db:"origin_label"`
	ReleasedTo  string     `json:"released_to" db:"released_to"`
	Notes       *string    `json:"notes" db:"notes"`
	Status      string     `json:"status" db:"status"`
	CreatedBy   int        `json:"created_by" db:"created_by"`
	CreatedByName *string  `json:"created_by_name,omitempty" db:"created_by_name"`
	CompletedAt *time.Time `json:"completed_at" db:"completed_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

type ReleaseDetail struct {
	Release
	Assets  []ReleaseAsset `json:"assets"`
	Stocks  []ReleaseStock `json:"stocks"`
	Summary ReleaseSummary `json:"summary"`
}

type ReleaseSummary struct {
	TotalAssets       int `json:"total_assets"`
	TotalStockQuantity int `json:"total_stock_quantity"`
}

type ReleaseAsset struct {
	ID           int     `json:"id" db:"id"`
	ReleaseID    int     `json:"-" db:"release_id"`
	ItemID       int     `json:"item_id" db:"item_id"`
	PyrCode      *string `json:"pyr_code" db:"pyr_code"`
	ItemSerial   *string `json:"item_serial" db:"item_serial"`
	CategoryName *string `json:"category_name" db:"category_name"`
	OriginLabel  *string `json:"origin_label" db:"origin_label"`
	LocationName *string `json:"location_name" db:"location_name"`
}

type ReleaseStock struct {
	ID             int     `json:"id" db:"id"`
	ReleaseID      int     `json:"-" db:"release_id"`
	StockID        int     `json:"stock_id" db:"stock_id"`
	ItemCategoryID int     `json:"item_category_id" db:"item_category_id"`
	CategoryName   *string `json:"category_name" db:"category_name"`
	Quantity       int     `json:"quantity" db:"quantity"`
	OriginLabel    *string `json:"origin_label" db:"origin_label"`
	LocationName   *string `json:"location_name" db:"location_name"`
}

// API request/response types

type CreateReleaseRequest struct {
	OriginID   *int              `json:"origin_id"`
	ReleasedTo string            `json:"released_to" binding:"required"`
	Notes      *string           `json:"notes"`
	Assets     []int             `json:"assets"`
	Stocks     []StockReleaseReq `json:"stocks"`
}

type StockReleaseReq struct {
	StockID  int `json:"stock_id" binding:"required"`
	Quantity int `json:"quantity" binding:"required,min=1"`
}

type UpdateItemsRequest struct {
	Assets []int             `json:"assets"`
	Stocks []StockReleaseReq `json:"stocks"`
}

type SuggestResponse struct {
	Assets []SuggestedAsset `json:"assets"`
	Stocks []SuggestedStock `json:"stocks"`
}

type SuggestedAsset struct {
	ID           int     `json:"id" db:"id"`
	PyrCode      *string `json:"pyr_code" db:"pyr_code"`
	ItemSerial   *string `json:"item_serial" db:"item_serial"`
	Status       string  `json:"status" db:"status"`
	CategoryName *string `json:"category_name" db:"category_name"`
	OriginLabel  *string `json:"origin_label" db:"origin_label"`
	LocationName *string `json:"location_name" db:"location_name"`
}

type SuggestedStock struct {
	ID           int     `json:"id" db:"id"`
	Quantity     int     `json:"quantity" db:"quantity"`
	CategoryName *string `json:"category_name" db:"category_name"`
	OriginLabel  *string `json:"origin_label" db:"origin_label"`
	LocationName *string `json:"location_name" db:"location_name"`
}
