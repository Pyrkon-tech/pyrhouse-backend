package models

import (
	"time"
)

type Transfer struct {
	ID           int      `json:"id"`
	FromLocation Location `json:"from_location"`
	ToLocation   Location `json:"to_location"`
	// ItemCollection       []interface{} `json:"items,omitempty"`
	AssetsCollection     []Asset     `json:"assets,omitempty"`
	StockItemsCollection []StockItem `json:"stock_items,omitempty"`
	TransferDate         time.Time   `json:"transfer_date"`
	Status               string      `json:"status"`
}

func (t *Transfer) CreateLogView() AuditLog {
	return AuditLog{
		ResourceID:   t.ID,
		ResourceType: "transfer",
	}
}
