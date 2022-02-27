package models

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

type PasswordReset struct {
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
}

type Role string

const (
	Admin    Role = "ADMIN"
	Customer Role = "CUSTOMER"
)
