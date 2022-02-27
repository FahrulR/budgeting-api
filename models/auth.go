package models

import (
	"time"
)

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  `json:"user"`
}

type RedisPayload struct {
	User         `json:"user"`
	RefreshToken string `json:"refresh-token"`
}

type User struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
}

type PasswordReset struct {
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
}

type Role string

const (
	Admin    Role = "ADMIN"
	Customer Role = "CUSTOMER"
)
