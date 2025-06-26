package transfers

type RemoveItemFromTransferRequest struct {
	ID         int `uri:"id" binding:"required"`
	ItemID     int `uri:"item_id" binding:"required"`
	LocationID int `json:"location_id"`
}
