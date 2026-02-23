package settings

import (
	"time"
)

type AppSettings struct {
	Key         string     `json:"key" db:"key"`
	Value       *string    `json:"value" db:"value"`
	Description string     `json:"description" db:"description"`
	UpdatedAt   *time.Time `json:"updated_at" db:"updated_at"`
}
