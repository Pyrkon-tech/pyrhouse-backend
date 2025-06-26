package models

type StockItem struct { // Non-Serializied Item
	ID       int          `json:"id,omitempty" db:"asset_id"`
	Category ItemCategory `json:"category" db:"category"`
	Location Location     `json:"location,omitempty"`
	Quantity int          `json:"quantity" db:"quantity"`
	Origin   string       `json:"origin"`
	Status   string       `json:"status,omitempty" db:"status"`
}

func (a StockItem) CreateLogView() AuditLog {
	return AuditLog{
		ResourceID:   a.ID,
		ResourceType: "stock",
	}
}

type StockItemFlat struct {
	ID         int `db:"stock_id"`
	CategoryID int `db:"category_id"`
	Quantity   int `db:"quantity"`
}

type FlatStockRecord struct {
	ID                    int     `db:"stock_id"`
	Quantity              int     `db:"quantity"`
	LocationID            int     `db:"location_id"`
	LocationName          string  `db:"location_name"`
	LocationPavilion      *string `db:"location_pavilion"`
	CategoryID            int     `db:"category_id"`
	CategoryType          string  `db:"category_type"`
	CategoryLabel         string  `db:"category_label"`
	CategoryPyrId         string  `db:"category_pyr_id"`
	CategoryEquipmentType string  `db:"category_equipment_type"`
	Origin                string  `db:"origin"`
	TransferStockID       int     `db:"transfer_stock_id"`
}
