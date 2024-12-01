package models

import (
	"time"
)

type Transfer struct {
	ID                   int         `json:"id"`
	FromLocation         Location    `json:"from_location"`
	ToLocation           Location    `json:"to_location"`
	AssetsCollection     []Asset     `json:"assets"`
	StockItemsCollection []StockItem `json:"stock_items"`
	TransferDate         time.Time   `json:"transfer_date"`
	Status               string      `json:"status"`
}

func (t *Transfer) CreateLogView() AuditLog {
	return AuditLog{
		ResourceID:   t.ID,
		ResourceType: "transfer",
	}
}
