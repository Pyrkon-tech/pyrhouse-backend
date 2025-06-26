package models

// CreateStockItemRequest reprezentuje żądanie utworzenia elementu magazynowego
type CreateStockItemRequest struct {
	CategoryID int    `json:"category_id" binding:"required"`
	LocationID int    `json:"location_id"`
	Quantity   int    `json:"quantity" binding:"required"`
	Origin     string `json:"origin"`
}

// PatchStockItemRequest reprezentuje żądanie aktualizacji elementu magazynowego
type PatchStockItemRequest struct {
	ID         int     `uri:"id" binding:"required"`
	LocationID *int    `json:"location_id"`
	Quantity   *int    `json:"quantity"`
	Origin     *string `json:"origin"`
}

// RemoveStockItemFromTransferRequest reprezentuje żądanie usunięcia elementu z transferu
type RemoveStockItemFromTransferRequest struct {
	Quantity     int `json:"quantity" binding:"required"`
	ToLocationID int `json:"location_id" binding:"required"`
	TransferID   int
	CategoryID   int
}

// MoveStockItemToLocationRequest reprezentuje żądanie przeniesienia elementu
type MoveStockItemToLocationRequest struct {
	Quantity       int `json:"quantity" binding:"required"`
	CategoryID     int
	FromLocationID int
	ToLocationID   int
}
