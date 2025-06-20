package models

import (
	"database/sql"
	"warehouse/pkg/metadata"
)

const (
	AssetStatusInStock   string = "in_stock"
	AssetStatusInTransit string = "in_transit"
	AssetStatusDelivered string = "delivered"
)

type Asset struct {
	ID       int             `json:"id" db:"asset_id"`
	Serial   *string         `json:"serial" db:"item_serial"`
	Location Location        `json:"location,omitempty"`
	Category ItemCategory    `json:"category"`
	Status   metadata.Status `json:"status"`
	PyrCode  string          `json:"pyrcode"`
	Origin   metadata.Origin `json:"origin"`
}

type FlatAssetRecord struct {
	ID                    int            `db:"asset_id"`
	Serial                sql.NullString `db:"item_serial"`
	Status                string         `db:"status"`
	Origin                string         `db:"origin"`
	PyrCode               sql.NullString `db:"pyr_code"`
	LocationId            int            `db:"location_id"`
	LocationName          string         `db:"location_name"`
	LocationPavilion      sql.NullString `db:"location_pavilion"`
	CategoryId            int            `db:"category_id"`
	CategoryType          string         `db:"category_type"`
	CategoryLabel         string         `db:"category_label"`
	CategoryPyrId         string         `db:"category_pyr_id"`
	CategoryEquipmentType string         `db:"category_equipment_type"`
}

func (fa *FlatAssetRecord) TransformToAsset() Asset {
	status, _ := metadata.NewStatus(fa.Status)
	origin, _ := metadata.NewOrigin(fa.Origin)

	var serial *string
	if fa.Serial.Valid {
		serial = &fa.Serial.String
	}

	var pavilion *string
	if fa.LocationPavilion.Valid {
		pavilion = &fa.LocationPavilion.String
	}

	return Asset{
		ID:      fa.ID,
		Serial:  serial,
		Status:  status,
		PyrCode: fa.PyrCode.String,
		Origin:  origin,
		Location: Location{
			ID:       fa.LocationId,
			Name:     fa.LocationName,
			Pavilion: pavilion,
		},
		Category: ItemCategory{
			ID:    fa.CategoryId,
			Name:  fa.CategoryType,
			Label: fa.CategoryLabel,
			PyrID: fa.CategoryPyrId,
			Type:  fa.CategoryEquipmentType,
		},
	}
}

func (a *Asset) CreateLogView() AuditLog {
	return AuditLog{
		ResourceID:   a.ID,
		ResourceType: "asset",
	}
}
