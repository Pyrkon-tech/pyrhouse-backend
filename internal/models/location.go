package models

const DefaultEquipmentLocationID = 1

type Location struct {
	ID       int     `json:"id" db:"id"`
	Name     string  `json:"name" db:"name"`
	Pavilion *string `json:"pavilion" db:"pavilion"`
	Details  *string `json:"details" db:"details"`
}
