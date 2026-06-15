package user

import (
	"context"
	"fmu-backend/internal/errs"

	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	Create(ctx context.Context, full_name, email, password, role string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByProvider(ctx context.Context, provider, providerID string) (*User, error)
	CreateWithOAuth(ctx context.Context, fullName, email, provider, providerID, avatar string) (*User, error)
}

type userService struct {
	userRepo UserRepository
}

func NewUserService(userRepo UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) Create(ctx context.Context, full_name, email, password, role string) (*User, error) {
	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		return nil, errs.ErrUserAlreadyExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return s.userRepo.Create(ctx, full_name, email, string(hashed), role)
}

func (s *userService) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

func (s *userService) GetByID(ctx context.Context, id string) (*User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *userService) GetByProvider(ctx context.Context, provider, providerID string) (*User, error) {
	return s.userRepo.GetByProvider(ctx, provider, providerID)
}

func (s *userService) CreateWithOAuth(ctx context.Context, fullName, email, provider, providerID, avatar string) (*User, error) {
	return s.userRepo.CreateWithOAuth(ctx, fullName, email, provider, providerID, avatar)
}
