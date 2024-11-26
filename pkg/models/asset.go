package models

import (
	"encoding/json"
	"fmt"
)

type Asset struct {
	ID          int                `json:"id" db:"asset_id"`
	Serial      string             `json:"serial" db:"item_serial"`
	Location    Location           `json:"location,omitempty"`
	Category    ItemCategory       `json:"category"`
	Status      string             `json:"status"`
	PyrCode     string             `json:"pyrcode"`
	Accessories []AssetAccessories `json:"accessories" db:"accessories"`
}

type AssetAccessories struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}

type FlatAssetRecord struct {
	ID            int    `db:"asset_id"`
	Serial        string `db:"item_serial"`
	Status        string `db:"status"`
	PyrCode       string `db:"pyr_code"`
	Accessories   []byte `db:"accessories"`
	LocationId    int    `db:"location_id"`
	LocationName  string `db:"location_name"`
	CategoryId    int    `db:"category_id"`
	CategoryType  string `db:"category_type"`
	CategoryLabel string `db:"category_label"`
	CategoryPyrId string `db:"category_pyr_id"`
}

func (fa *FlatAssetRecord) TransformToAsset() (Asset, error) {
	var accessories []AssetAccessories
	if err := json.Unmarshal(fa.Accessories, &accessories); err != nil {
		return Asset{}, fmt.Errorf("failed to unmarshal accessories: %w", err)
	}

	return Asset{
		ID:          fa.ID,
		Serial:      fa.Serial,
		Status:      fa.Status,
		PyrCode:     fa.PyrCode,
		Accessories: accessories,
		Location: Location{
			ID:   fa.LocationId,
			Name: fa.LocationName,
		},
		Category: ItemCategory{
			ID:    fa.CategoryId,
			Type:  fa.CategoryType,
			Label: fa.CategoryLabel,
			PyrID: fa.CategoryPyrId,
		},
	}, nil
}

func (a *Asset) CreateLogView() AuditLog {
	return AuditLog{
		ResourceID:   a.ID,
		ResourceType: "asset",
	}
}
