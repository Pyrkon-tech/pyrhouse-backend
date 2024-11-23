package models

type Asset struct {
	ID       int          `json:"id"`
	Serial   string       `json:"serial"`
	Location Location     `json:"location"`
	Status   string       `json:"status"`
	Category ItemCategory `json:"category"`
}

func (a *Asset) CreateLogView() AuditLog {
	return AuditLog{
		ResourceID:   a.ID,
		ResourceType: "asset",
	}
}
