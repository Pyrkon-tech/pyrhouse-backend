package models

import "warehouse/pkg/roles"

type User struct {
	ID           int        `json:"id" db:"id"`
	Username     string     `json:"username" db:"username"`
	Fullname     string     `json:"fullname" db:"fullname"`
	PasswordHash string     `json:"-" db:"password_hash"`
	Role         roles.Role `json:"role" db:"role"`
	Points       int        `json:"points" db:"points"`
}

type CreateUserRequest struct {
	Username string     `json:"username" binding:"required"`
	Password string     `json:"password" binding:"required"`
	Fullname string     `json:"fullname"`
	Role     roles.Role `json:"role" binding:"required"`
	Points   int        `json:"points"`
}

type UpdateUserRequest struct {
	Fullname *string     `json:"fullname"`
	Password *string     `json:"password"`
	Role     *roles.Role `json:"role"`
	Points   *int        `json:"points"`
}
