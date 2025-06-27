package transfers

import (
	"time"
)

// Transfer reprezentuje transfer w systemie
type Transfer struct {
	ID                 string            `json:"id" db:"id"`
	Type               string            `json:"type" db:"type"`
	Status             string            `json:"status" db:"status"`
	SourceLocationID   *string           `json:"source_location_id,omitempty" db:"source_location_id"`
	TargetLocationID   *string           `json:"target_location_id,omitempty" db:"target_location_id"`
	DeliveryLocationID *string           `json:"delivery_location_id,omitempty" db:"delivery_location_id"`
	CreatedBy          string            `json:"created_by" db:"created_by"`
	CreatedAt          time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at" db:"updated_at"`
	Metadata           map[string]string `json:"metadata,omitempty" db:"metadata"`
}

// TransferRequest reprezentuje żądanie utworzenia transferu
type TransferRequest struct {
	Type               string            `json:"type" binding:"required"`
	SourceLocationID   *string           `json:"source_location_id,omitempty"`
	TargetLocationID   *string           `json:"target_location_id,omitempty"`
	DeliveryLocationID *string           `json:"delivery_location_id,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
}

// TransferResponse reprezentuje odpowiedź z danymi transferu
type TransferResponse struct {
	ID                   string            `json:"id"`
	Type                 string            `json:"type"`
	Status               string            `json:"status"`
	SourceLocationID     *string           `json:"source_location_id,omitempty"`
	SourceLocationName   *string           `json:"source_location_name,omitempty"`
	TargetLocationID     *string           `json:"target_location_id,omitempty"`
	TargetLocationName   *string           `json:"target_location_name,omitempty"`
	DeliveryLocationID   *string           `json:"delivery_location_id,omitempty"`
	DeliveryLocationName *string           `json:"delivery_location_name,omitempty"`
	CreatedBy            string            `json:"created_by"`
	CreatedByUsername    string            `json:"created_by_username,omitempty"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}
