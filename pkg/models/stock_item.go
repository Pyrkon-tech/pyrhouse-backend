package models

type StockItem struct { // Non-Serializied Item
	ID       int          `json:"id"`
	Category ItemCategory `json:"category"`
	Location Location     `json:"location"`
	Quantity int          `json:"quantity"`
}

func (a *StockItem) CreateLogView() AuditLog {
	return AuditLog{
		ResourceID:   a.ID,
		ResourceType: "stock",
	}
}
