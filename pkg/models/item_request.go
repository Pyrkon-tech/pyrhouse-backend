package models

type ItemRequest struct {
	ID         int    `json:"id"`
	Serial     string `json:"serial" binding:"required"`
	LocationId int    `json:"location_id" default:"1"`
	Status     string `json:"status"`
	CategoryId int    `json:"category_id" binding:"required"`
	Origin     string `json:"origin"`
}

type BulkItemRequest struct {
	Serials    []string `json:"serials" binding:"required,min=1"`
	LocationId int      `json:"location_id" default:"1"`
	Status     string   `json:"status"`
	CategoryId int      `json:"category_id" binding:"required"`
	Origin     string   `json:"origin"`
}
