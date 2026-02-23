package models

// CreateStockItemRequest represents a request to create a stock item
type CreateStockItemRequest struct {
	CategoryID   int     `json:"category_id" binding:"required"`
	LocationID   int     `json:"location_id"`
	Quantity     int     `json:"quantity" binding:"required"`
	Origin       string  `json:"origin"`
	OriginID     *int    `json:"-"`
	OriginSuffix *string `json:"-"`
}

// PatchStockItemRequest represents a request to update a stock item
type PatchStockItemRequest struct {
	ID           int     `uri:"id" binding:"required"`
	LocationID   *int    `json:"location_id"`
	Quantity     *int    `json:"quantity"`
	Origin       *string `json:"origin"`
	OriginID     *int    `json:"-"`
	OriginSuffix *string `json:"-"`
}

// RemoveStockItemFromTransferRequest represents a request to remove an item from a transfer
type RemoveStockItemFromTransferRequest struct {
	Quantity     int `json:"quantity" binding:"required"`
	ToLocationID int `json:"location_id" binding:"required"`
	TransferID   int
	CategoryID   int
}

// MoveStockItemToLocationRequest represents a request to move an item to a location
type MoveStockItemToLocationRequest struct {
	Quantity       int `json:"quantity" binding:"required"`
	CategoryID     int
	FromLocationID int
	ToLocationID   int
}
