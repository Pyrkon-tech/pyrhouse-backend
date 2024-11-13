package models

type ItemCategory struct {
	ID    int    `json:"id"`
	Type  string `json:"type" binding:"required,alphanum"`
	Label string `json:"label" binding:"required"`
}
