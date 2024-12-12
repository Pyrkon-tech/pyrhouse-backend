package models

type UnserializedItemRequest struct {
	ItemCategoryID int `json:"category_id" binding:"required"`
	Quantity       int `json:"quantity" binding:"required"`
}

type TransferItemRequest struct {
	ID       int `json:"id" binding:"required"`
	Quantity int `json:"quantity" binding:"omitempty"`
}

type TransferRequest struct {
	TransferID          int
	FromLocationID      int                   `json:"from_location_id" binding:"required"`
	LocationID          int                   `json:"location_id" binding:"required"`
	ItemCollection      []TransferItemRequest `json:"items"`
	AssetItemCollection []int
	StockItemCollection []UnserializedItemRequest
}
