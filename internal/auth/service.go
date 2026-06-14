package auth

import (
	"context"
	"fmu-backend/internal/user"
)

type AuthService interface {
	Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error)
}

type authService struct {
	userService user.UserService
}

func NewAuthService(userService user.UserService) AuthService {
	return &authService{
		userService: userService,
	}
}

func (s *authService) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	u, err := s.userService.Create(ctx, req.FullName, req.Email, req.Password, "student")
	if err != nil {
		return nil, err
	}

	return &RegisterResponse{
		ID:        u.ID,
		FullName:  u.FullName,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
	}, nil
}
