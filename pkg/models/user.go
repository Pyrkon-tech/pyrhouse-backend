package models

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	Fullname     string `json:"fullname"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
}
