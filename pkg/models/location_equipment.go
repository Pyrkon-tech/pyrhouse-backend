package models

type LocationEquipment struct {
	Assets     []Asset     `json:"assets"`
	StockItems []StockItem `json:"stock_items"`
}
