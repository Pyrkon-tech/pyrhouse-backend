package models

import "time"

type Transfer struct {
	ID               int               `json:"id"`
	FromLocationID   int               `json:"from_location_id"`
	FromLocationName string            `json:"from_location_name"`
	ToLocationID     int               `json:"to_location_id"`
	ToLocationName   string            `json:"to_location_name"`
	Status           string            `json:"status"`
	TransferDate     time.Time         `json:"transfer_date"`
	DeliveryLocation *DeliveryLocation `json:"delivery_location,omitempty"`
}

type DeliveryLocation struct {
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	Timestamp time.Time `json:"timestamp"`
}
