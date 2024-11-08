package models

type Item struct {
	ID         int    `json:"id"`
	Type       string `json:"type"`
	Serial     string `json:"serial"`
	LocationId int    `json:"location_id"`
	Status     string `json:"status"`
}
