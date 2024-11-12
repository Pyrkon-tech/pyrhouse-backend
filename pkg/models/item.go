package models

type Item struct {
	ID       int          `json:"id"`
	Serial   string       `json:"serial"`
	Location Location     `json:"location"`
	Status   string       `json:"status"`
	Category ItemCategory `json:"category"`
}
