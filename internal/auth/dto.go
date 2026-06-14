package auth

import "time"

type (
	RegisterRequest struct {
		FullName string `json:"full_name" validate:"required,min=2,max=100"`
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8"`
	}

	RegisterResponse struct {
		ID        string    `json:"id"`
		FullName  string    `json:"full_name"`
		Email     string    `json:"email"`
		Role      string    `json:"role"`
		CreatedAt time.Time `json:"created_at"`
	}
)
