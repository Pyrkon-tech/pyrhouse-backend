package models

const DefaultEquipmentLocationID = 1

type Location struct {
	ID   int    `json:"id,omitempty" db:"id"`
	Name string `json:"name,omitempty" db:"name"`
}
