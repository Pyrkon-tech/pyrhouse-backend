package transfers

type UnserializedItemRequest struct {
	ItemID   int `json:"id" binding:"required"`
	Quantity int `json:"quantity" binding:"required"`
}

type TransferRequest struct {
	LocationID                 int                       `json:"location_id" binding:"required"`
	SerialziedItemCollection   []int                     `json:"serialized_item_collection"`
	UnserializedItemCollection []UnserializedItemRequest `json:"unserialized_item_collection"`
}
