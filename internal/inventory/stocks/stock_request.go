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
	LocationID int `json:"location_id" binding:"required"`
	Quantity   int `json:"quantity" binding:"required"`
	TransferID int
	CategoryID int
}
