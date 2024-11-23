package models

type ItemCategory struct {
	ID    int    `json:"id"`
	Type  string `json:"type,omitempty" binding:"required,alphanum"`
	Label string `json:"label,omitempty" binding:"required"`
}
