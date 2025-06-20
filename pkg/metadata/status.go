package metadata

import "fmt"

type Status string

const (
	StatusInStock     Status = "in_stock" // deprecated
	StatusInTransit   Status = "in_transit"
	StatusLocated     Status = "located"
	StatusCompleted   Status = "completed"
	StatusAvailable   Status = "available"
	StatusUnavailable Status = "unavailable"
	StatusCancelled   Status = "cancelled"
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
	case StatusInStock, StatusInTransit, StatusLocated, StatusCompleted, StatusAvailable, StatusUnavailable, StatusCancelled:
		return true
	default:
		return false
	}
}
