package models

type ItemCategory struct {
	ID    int    `json:"id,omitempty" db:"category_id"`
	Type  string `json:"type,omitempty" binding:"required,alphanum" db:"type"`
	Label string `json:"label,omitempty" binding:"required" db:"label"`
	PyrID string `json:"pyr_id" binding:"omitempty,alphanum,min=1,max=3" db:"pyr_id"`
}
