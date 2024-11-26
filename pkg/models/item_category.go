package models

type ItemCategory struct {
	ID          int               `json:"id,omitempty" db:"category_id"`
	Type        string            `json:"type,omitempty" binding:"required,alphanum" db:"type"`
	Label       string            `json:"label,omitempty" binding:"required" db:"label"`
	PyrID       string            `json:"pyr_id" binding:"alphanum,min=1,max=2" db:"pyr_id"`
	Accessories []ItemAccessories `json:"accessories" db:"accessories"`
}

type ItemAccessories struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}
