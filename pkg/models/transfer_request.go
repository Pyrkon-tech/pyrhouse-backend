package models

type StockItemRequest struct {
	ID       int `json:"id" binding:"required"`
	Quantity int `json:"quantity" binding:"omitempty,required,gte=1"`
}

type AssetItemRequest struct {
	ID int `json:"id" binding:"required"`
}

type TransferRequest struct {
	TransferID          int
	FromLocationID      int                `json:"from_location_id" binding:"required"`
	LocationID          int                `json:"location_id" binding:"required"`
	AssetItemCollection []AssetItemRequest `json:"assets"`
	StockItemCollection []StockItemRequest `json:"stocks"`
}
