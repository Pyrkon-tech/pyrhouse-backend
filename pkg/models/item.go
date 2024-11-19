package models

type Asset struct { // assets table
	ID       int          `json:"id"`
	Serial   string       `json:"serial"`
	Location Location     `json:"location"`
	Status   string       `json:"status"`
	Category ItemCategory `json:"category"`
}
