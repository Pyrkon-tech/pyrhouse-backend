package models

type StockItem struct { // Non-Serializied Item
	ID       int
	Category ItemCategory
	Quantity int
}
