package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"fmu-backend/internal/db/sqlc"
)

type UserRepository interface {
	Create(ctx context.Context, full_name, email, password, role string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByProvider(ctx context.Context, provider, providerID string) (*User, error)
	CreateWithOAuth(ctx context.Context, fullName, email, provider, providerID, avatar string) (*User, error)
}

type userRepository struct {
	queries *sqlc.Queries
}

func NewUserRepository(queries *sqlc.Queries) UserRepository {
	return &userRepository{
		queries: queries,
	}
}

func (r *userRepository) Create(ctx context.Context, full_name, email, password, role string) (*User, error) {
	row, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		FullName: full_name,
		Email:    email,
		Password: &password,
		Role:     role,
	})
	if err != nil {
		return nil, err
	}
	return toDomainUser(row), nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toDomainUser(row), nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*User, error) {
	row, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toDomainUser(row), nil
}

func (r *userRepository) GetByProvider(ctx context.Context, provider, providerID string) (*User, error) {
	row, err := r.queries.GetUserByProvider(ctx, sqlc.GetUserByProviderParams{
		Provider:   &provider,
		ProviderID: &providerID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toDomainUser(row), nil
}

func (r *userRepository) CreateWithOAuth(ctx context.Context, fullName, email, provider, providerID, avatar string) (*User, error) {
	row, err := r.queries.CreateUserWithOAuth(ctx, sqlc.CreateUserWithOAuthParams{
		FullName:   fullName,
		Email:      email,
		Provider:   &provider,
		ProviderID: &providerID,
		Avatar:     &avatar,
	})
	if err != nil {
		return nil, err
	}
	return toDomainUser(row), nil
}

func toDomainUser(u sqlc.User) *User {
	return &User{
		ID:            u.ID,
		FullName:      u.FullName,
		Email:         u.Email,
		Password:      u.Password,
		Provider:      u.Provider,
		ProviderID:    u.ProviderID,
		Avatar:        u.Avatar,
		EmailVerified: u.EmailVerified,
		Role:          u.Role,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}
