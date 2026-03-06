package metadata

import "fmt"

type Status string

const (
	StatusAvailable   Status = "available"   // Asset is at its location, ready for operations
	StatusInTransit   Status = "in_transit"  // Asset is being transferred between locations
	StatusUnavailable Status = "unavailable" // Asset is temporarily unavailable (e.g. damaged)
)

func NewStatus(value string) (Status, error) {
	status := Status(value)
	if !status.isValid() {
		return "", fmt.Errorf("invalid status: %s", value)
	}
	return status, nil
}

func (s Status) isValid() bool {
	switch s {
	case StatusAvailable, StatusInTransit, StatusUnavailable:
		return true
	default:
		return false
	}
}
