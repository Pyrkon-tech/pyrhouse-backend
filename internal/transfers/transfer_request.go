package transfers

type RemoveItemFromTransferRequest struct {
	ID         int `uri:"id" binding:"required"`
	ItemID     int `uri:"item_id" binding:"required"`
	LocationID int `json:"location_id"`
}

// type RemoveStockItemFromTransferRequest struct {
// 	LocationID int `json:"location_id" binding:"required"`
// 	Quantity   int `json:"quantity" binding:"required"`
// 	TransferID int
// 	CategoryID int
// }
