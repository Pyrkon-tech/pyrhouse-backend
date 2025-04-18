package models

import (
	"time"
)

type Transfer struct {
	ID           int      `json:"id"`
	FromLocation Location `json:"from_location"`
	ToLocation   Location `json:"to_location"`
	// ItemCollection       []interface{} `json:"items,omitempty"`
	AssetsCollection     []Asset           `json:"assets,omitempty"`
	StockItemsCollection []StockItem       `json:"stock_items,omitempty"`
	TransferDate         time.Time         `json:"transfer_date"`
	Status               string            `json:"status"`
	Users                []User            `json:"users,omitempty"`
	DeliveryLocation     *DeliveryLocation `json:"delivery_location,omitempty"`
}

type DeliveryLocation struct {
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	Timestamp time.Time `json:"timestamp"`
}

type TransferUser struct {
	UserID int `json:"id" binding:"required" db:"user_id"`
}

func (tu *TransferUser) CreateLogView() AuditLog {
	return AuditLog{
		ResourceID:   tu.UserID,
		ResourceType: "user",
	}
}

func (t *Transfer) CreateLogView() AuditLog {
	return AuditLog{
		ResourceID:   t.ID,
		ResourceType: "transfer",
	}
}
