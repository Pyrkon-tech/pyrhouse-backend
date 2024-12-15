package models

type ItemCategory struct {
	ID    int    `json:"id,omitempty" db:"category_id"`
	Name  string `json:"name,omitempty" binding:"required" db:"type"`
	Label string `json:"label,omitempty" binding:"required" db:"label"`
	PyrID string `json:"pyr_id" binding:"omitempty,alphanum,min=1,max=3" db:"pyr_id"`
	Type  string `json:"type" binding:"alphanum,min=1,max=24" db:"category_type"`
}
