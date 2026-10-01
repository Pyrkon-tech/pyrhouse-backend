package metadata

type Status string

const (
	StatusAvailable   Status = "available"   // Asset is at its location, ready for operations
	StatusInTransit   Status = "in_transit"  // Asset is being transferred between locations
	StatusUnavailable Status = "unavailable" // Asset is temporarily unavailable (e.g. damaged)
)
