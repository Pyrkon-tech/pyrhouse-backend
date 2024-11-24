package models

type StockItem struct { // Non-Serializied Item
	ID       int          `json:"id,omitempty" db:"asset_id"`
	Category ItemCategory `json:"category" db:"category"`
	Location Location     `json:"location,omitempty"`
	Quantity int          `json:"quantity" db:"quantity"`
}

func (a StockItem) CreateLogView() AuditLog {
	return AuditLog{
		ResourceID:   a.ID,
		ResourceType: "stock",
	}
}

type StockItemFlat struct {
	CategoryID int `db:"category_id"`
	Quantity   int `db:"quantity"`
}
