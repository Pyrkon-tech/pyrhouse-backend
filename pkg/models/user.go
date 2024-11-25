package models

type User struct {
	ID           int    `json:"id" db:"id"`
	Username     string `json:"username" db:"username"`
	Fullname     string `json:"fullname" db:"fullname"`
	PasswordHash string `json:"-"`
	Role         string `json:"role" db:"role"`
}

type UserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Fullname string `json:"fullname"`
	Role     string `json:"role" binding:"required"`
}
