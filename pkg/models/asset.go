package models

import (
	"database/sql"
	"warehouse/internal/metadata"
)

const (
	AssetStatusInStock   string = "in_stock"
	AssetStatusInTransit string = "in_transit"
	AssetStatusDelivered string = "delivered"
)

type Asset struct {
	ID       int             `json:"id" db:"asset_id"`
	Serial   string          `json:"serial" db:"item_serial"`
	Location Location        `json:"location,omitempty"`
	Category ItemCategory    `json:"category"`
	Status   string          `json:"status"`
	PyrCode  string          `json:"pyrcode"`
	Origin   metadata.Origin `json:"origin"`
}

type FlatAssetRecord struct {
	ID            int            `db:"asset_id"`
	Serial        string         `db:"item_serial"`
	Status        string         `db:"status"`
	Origin        string         `db:"origin"`
	PyrCode       sql.NullString `db:"pyr_code"`
	LocationId    int            `db:"location_id"`
	LocationName  string         `db:"location_name"`
	CategoryId    int            `db:"category_id"`
	CategoryType  string         `db:"category_type"`
	CategoryLabel string         `db:"category_label"`
	CategoryPyrId string         `db:"category_pyr_id"`
}

func (fa *FlatAssetRecord) TransformToAsset() Asset {
	origin, _ := metadata.NewOrigin(fa.Origin)

	return Asset{
		ID:      fa.ID,
		Serial:  fa.Serial,
		Status:  fa.Status,
		PyrCode: fa.PyrCode.String,
		Origin:  origin,
		Location: Location{
			ID:   fa.LocationId,
			Name: fa.LocationName,
		},
		Category: ItemCategory{
			ID:    fa.CategoryId,
			Name:  fa.CategoryType,
			Label: fa.CategoryLabel,
			PyrID: fa.CategoryPyrId,
		},
	}
}

func (a *Asset) CreateLogView() AuditLog {
	return AuditLog{
		ResourceID:   a.ID,
		ResourceType: "asset",
	}
}
