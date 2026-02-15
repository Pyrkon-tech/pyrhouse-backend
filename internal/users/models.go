package users

import (
	"time"
	"warehouse/internal/roles"
)

type User struct {
	ID           string            `json:"id" db:"id"`
	Username     string            `json:"username" db:"username"`
	PasswordHash string            `json:"-" db:"password_hash"`
	Role         roles.Role        `json:"role" db:"role"`
	Active       bool              `json:"active" db:"active"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
	Metadata     map[string]string `json:"metadata,omitempty" db:"metadata"`
}

type UserRequest struct {
	Username string            `json:"username" binding:"required"`
	Password string            `json:"password,omitempty"`
	Role     string            `json:"role" binding:"required"`
	Active   bool              `json:"active"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type UserResponse struct {
	ID        string            `json:"id"`
	Username  string            `json:"username"`
	Role      string            `json:"role"`
	Active    bool              `json:"active"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}
