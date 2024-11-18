package models

type UnserializedItemRequest struct {
	ItemCategoryID int `json:"category_id" binding:"required"`
	Quantity       int `json:"quantity" binding:"required"`
}

type TransferRequest struct {
	TransferID                 int
	FromLocationID             int                       `json:"from_location_id" binding:"required"`
	LocationID                 int                       `json:"location_id" binding:"required"`
	SerialziedItemCollection   []int                     `json:"serialized_item_collection"`
	UnserializedItemCollection []UnserializedItemRequest `json:"unserialized_item_collection"`
}
