package models

import "warehouse/pkg/roles"

type User struct {
	ID           int        `json:"id" db:"id"`
	Username     string     `json:"username" db:"username"`
	Fullname     string     `json:"fullname" db:"fullname"`
	PasswordHash string     `json:"-" db:"password_hash"`
	Role         roles.Role `json:"role" db:"role"`
	Points       int        `json:"points" db:"points"`
	Active       bool       `json:"active" db:"active"`
}

type CreateUserRequest struct {
	Username string      `json:"username" binding:"required"`
	Password string      `json:"password" binding:"required"`
	Fullname string      `json:"fullname"`
	Role     *roles.Role `json:"role,omitempty"`
	Points   int         `json:"points"`
	Active   bool        `json:"active"`
}

type UpdateUserRequest struct {
	Fullname *string     `json:"fullname"`
	Password *string     `json:"password"`
	Role     *roles.Role `json:"role"`
	Points   *int        `json:"points"`
	Username *string     `json:"username"`
	Active   *bool       `json:"active"`
}

// UserChanges reprezentuje pola użytkownika, które mogą być zmienione
type UserChanges struct {
	PasswordHash *string `db:"password_hash"`
	Role         *string `db:"role"`
	Points       *int    `db:"points"`
	Fullname     *string `db:"fullname"`
	Username     *string `db:"username"`
	Active       *bool   `db:"active"`
}

// HasChanges sprawdza, czy jakiekolwiek pole zostało zmienione
func (c *UserChanges) HasChanges() bool {
	return c.PasswordHash != nil || c.Role != nil || c.Points != nil || c.Fullname != nil || c.Username != nil || c.Active != nil
}
