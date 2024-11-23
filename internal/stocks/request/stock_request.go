package stock_request

type StockItemRequest struct {
	CategoryID int `json:"category_id" binding:"required"`
	LocationID int `json:"location_id"`
	Quantity   int `json:"quantity" binding:"required"`
}
