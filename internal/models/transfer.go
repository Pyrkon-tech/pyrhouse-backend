package models

import (
	"time"
)

// TransferSummary is a transfer without its items, as returned by list endpoints.
type TransferSummary struct {
	ID           int       `json:"id"`
	FromLocation Location  `json:"from_location"`
	ToLocation   Location  `json:"to_location"`
	TransferDate time.Time `json:"transfer_date"`
	Status       string    `json:"status"`
}

// Transfer is the full transfer (GET /transfers/:id); every collection is sent, empty or not.
type Transfer struct {
	TransferSummary
	AssetsCollection     []Asset               `json:"assets"`
	StockItemsCollection []StockItem           `json:"stock_items"`
	Users                []TransferParticipant `json:"users"`
	DeliveryLocation     *DeliveryLocation     `json:"delivery_location"`
}

// TransferParticipant is a user assigned to carry out a transfer.
type TransferParticipant struct {
	ID       int     `json:"id" db:"id"`
	Username string  `json:"username" db:"username"`
	Fullname *string `json:"fullname" db:"fullname"`
}

type DeliveryLocation struct {
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	Timestamp time.Time `json:"timestamp"`
}

type DeliveryLocationRequest struct {
	DeliveryLocation DeliveryLocation `json:"delivery_location" binding:"required"`
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
