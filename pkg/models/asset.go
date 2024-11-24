package models

type Asset struct {
	ID       int          `json:"id" db:"asset_id"`
	Serial   string       `json:"serial" db:"item_serial"`
	Location Location     `json:"location,omitempty"`
	Category ItemCategory `json:"category"`
}

func (a *Asset) CreateLogView() AuditLog {
	return AuditLog{
		ResourceID:   a.ID,
		ResourceType: "asset",
	}
}
