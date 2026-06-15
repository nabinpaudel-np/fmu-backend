package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	Create(ctx context.Context, full_name, email, password, role string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByProvider(ctx context.Context, provider, providerID string) (*User, error)
	CreateWithOAuth(ctx context.Context, fullName, email, provider, providerID, avatar string) (*User, error)
}

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) Create(ctx context.Context, full_name, email, password, role string) (*User, error) {
	query := `
	INSERT INTO users (full_name, email, password, role)
	VALUES ($1, $2, $3, $4)
	RETURNING id, full_name, email, role, created_at, updated_at
	`

	user := &User{}
	err := r.db.QueryRow(ctx, query, full_name, email, password, role).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil

}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, full_name, email, password, oauth_provider, oauth_id, avatar, email_verified, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	user := &User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Password,
		&user.Provider,
		&user.ProviderID,
		&user.Avatar,
		&user.EmailVerified,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, full_name, email, password, oauth_provider, oauth_id, avatar, email_verified, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	user := &User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Password,
		&user.Provider,
		&user.ProviderID,
		&user.Avatar,
		&user.EmailVerified,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (r *userRepository) GetByProvider(ctx context.Context, provider, providerID string) (*User, error) {
	query := `
		SELECT id, full_name, email, password, oauth_provider, oauth_id, avatar, email_verified, role, created_at, updated_at
		FROM users
		WHERE oauth_provider = $1 AND oauth_id = $2
	`
	user := &User{}
	err := r.db.QueryRow(ctx, query, provider, providerID).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Password,
		&user.Provider,
		&user.ProviderID,
		&user.Avatar,
		&user.EmailVerified,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (r *userRepository) CreateWithOAuth(ctx context.Context, fullName, email, provider, providerID, avatar string) (*User, error) {
	query := `
		INSERT INTO users (full_name, email, oauth_provider, oauth_id, avatar, email_verified, role)
		VALUES ($1, $2, $3, $4, $5, true, 'student')
		RETURNING id, full_name, email, oauth_provider, oauth_id, avatar, email_verified, role, created_at, updated_at
	`

	user := &User{}
	err := r.db.QueryRow(ctx, query, fullName, email, provider, providerID, avatar).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.Provider,
		&user.ProviderID,
		&user.Avatar,
		&user.EmailVerified,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}
