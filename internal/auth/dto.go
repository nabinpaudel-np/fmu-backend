package auth

import "time"

type (
	RegisterRequest struct {
		FullName string `json:"full_name" validate:"required,min=2,max=100"`
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8"`
	}

	RegisterResponse struct {
		UserID    string    `json:"user_id"`
		FullName  string    `json:"full_name"`
		Email     string    `json:"email"`
		Role      string    `json:"role"`
		CreatedAt time.Time `json:"created_at"`
	}
)

type (
	LoginRequest struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=6"`
	}

	LoginResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		UserID       string `json:"user_id"`
		FullName     string `json:"full_name"`
		Email        string `json:"email"`
		Avatar       string `json:"avatar,omitempty"`
	}
)

type (
	RefreshRequest struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	RefreshResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		UserID       string `json:"user_id"`
		FullName     string `json:"full_name"`
		Email        string `json:"email"`
		Avatar       string `json:"avatar,omitempty"`
	}
)

type (
	LogoutRequest struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
)
