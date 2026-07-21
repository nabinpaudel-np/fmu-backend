package auth

import "time"

type (
	RegisterRequest struct {
		FullName string `json:"full_name" validate:"required,min=2,max=100" example:"Ada Lovelace"`
		Email    string `json:"email" validate:"required,email" example:"ada@example.com"`
		Password string `json:"password" validate:"required,min=8" example:"correct-horse-battery-staple"`
	}

	RegisterResponse struct {
		UserID    string    `json:"user_id" example:"d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b"`
		FullName  string    `json:"full_name" example:"Ada Lovelace"`
		Email     string    `json:"email" example:"ada@example.com"`
		Role      string    `json:"role" example:"user"`
		CreatedAt time.Time `json:"created_at" example:"2026-06-28T10:55:59Z"`
	}
)

type (
	LoginRequest struct {
		Email    string `json:"email" validate:"required,email" example:"ada@example.com"`
		Password string `json:"password" validate:"required,min=6" example:"correct-horse-battery-staple"`
	}

	LoginResponse struct {
		AccessToken  string `json:"-"`
		RefreshToken string `json:"-"`
		UserID       string `json:"user_id" example:"d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b"`
		FullName     string `json:"full_name" example:"Ada Lovelace"`
		Email        string `json:"email" example:"ada@example.com"`
		Avatar       string `json:"avatar,omitempty" example:"https://cdn.example.com/avatars/ada.png"`
		Role         string `json:"role" example:"student"`
		RepresentativeUniversityID string `json:"representative_university_id,omitempty" example:"d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b"`
	}
)

type (
	RefreshRequest struct{}

	RefreshResponse struct {
		AccessToken  string `json:"-"`
		RefreshToken string `json:"-"`
		UserID       string `json:"user_id" example:"d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b"`
		FullName     string `json:"full_name" example:"Ada Lovelace"`
		Email        string `json:"email" example:"ada@example.com"`
		Avatar       string `json:"avatar,omitempty" example:"https://cdn.example.com/avatars/ada.png"`
		Role         string `json:"role" example:"student"`
		RepresentativeUniversityID string `json:"representative_university_id,omitempty" example:"d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b"`
	}
)

type LogoutRequest struct{}

type MeResponse struct {
	UserID       string `json:"user_id" example:"d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b"`
	FullName     string `json:"full_name" example:"Ada Lovelace"`
	Email        string `json:"email" example:"ada@example.com"`
	Avatar       string `json:"avatar,omitempty" example:"https://cdn.example.com/avatars/ada.png"`
	Role         string `json:"role" example:"student"`
	RepresentativeUniversityID string `json:"representative_university_id,omitempty" example:"d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b"`
}
