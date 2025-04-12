package stocks

type StockItemRequest struct {
	CategoryID int    `json:"category_id" binding:"required"`
	LocationID int    `json:"location_id"`
	Quantity   int    `json:"quantity" binding:"required"`
	Origin     string `json:"origin"`
}

type PatchStockItemRequest struct {
	ID         int     `uri:"id" binding:"required"`
	LocationID *int    `json:"location_id"`
	Quantity   *int    `json:"quantity"`
	Origin     *string `json:"origin"`
}

type RemoveStockItemFromTransferRequest struct {
	Quantity     int `json:"quantity" binding:"required"`
	ToLocationID int `json:"location_id" binding:"required"`
	TransferID   int
	CategoryID   int
}

type MoveStockItemToLocationRequest struct {
	Quantity       int `json:"quantity" binding:"required"`
	CategoryID     int
	FromLocationID int
	ToLocationID   int
}
