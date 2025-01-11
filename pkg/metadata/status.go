package metadata

import "fmt"

type Status string

const (
	StatusInTransit   Status = "in_transit"
	StatusCompleted   Status = "completed"
	StatusConfirmed   Status = "confirmed"
	StatusAvailable   Status = "available"
	StatusUnavailable Status = "unavailable"
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
	case StatusInTransit, StatusCompleted, StatusConfirmed, StatusAvailable, StatusUnavailable:
		return true
	default:
		return false
	}
}
