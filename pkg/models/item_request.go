package models

type ItemRequest struct {
	ID         int     `json:"id"`
	Serial     *string `json:"serial" binding:"omitempty"`
	LocationId int     `json:"location_id" default:"1"`
	Status     string  `json:"status"`
	CategoryId int     `json:"category_id" binding:"required"`
	Origin     string  `json:"origin"`
}

type BulkItemRequest struct {
	Serials    []*string `json:"serials" binding:"omitempty,min=1"`
	LocationId int       `json:"location_id" default:"1"`
	Status     string    `json:"status"`
	CategoryId int       `json:"category_id" binding:"required"`
	Origin     string    `json:"origin"`
}

type PatchItemCategoryRequest struct {
	ID    int     `uri:"id" binding:"required"`
	Label *string `json:"label"`
	Type  *string `json:"type"`
	PyrID *string `json:"pyr_id" binding:"omitempty,alphanum,min=1,max=3"`
}

type CreateAssetRequest struct {
	Serial     *string `json:"serial" binding:"omitempty"`
	LocationId int     `json:"location_id" default:"1"`
	Status     string  `json:"status"`
	CategoryId int     `json:"category_id" binding:"required"`
	Origin     string  `json:"origin"`
}

type EmergencyAssetRequest struct {
	Quantity   int    `json:"quantity" binding:"required,min=1"`
	LocationId int    `json:"location_id" default:"1"`
	Status     string `json:"status"`
	CategoryId int    `json:"category_id" binding:"required"`
	Origin     string `json:"origin"`
}
