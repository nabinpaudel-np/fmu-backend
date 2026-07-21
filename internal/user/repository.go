package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"fmu-backend/internal/db/sqlc"
)

type UserRepository interface {
	Create(ctx context.Context, full_name, email, password, role string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByProvider(ctx context.Context, provider, providerID string) (*User, error)
	CreateWithOAuth(ctx context.Context, fullName, email, provider, providerID, avatar string) (*User, error)
	CreateRepresentative(ctx context.Context, fullName, email, passwordHash, universityID string) (*User, error)
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

func (r *userRepository) CreateRepresentative(ctx context.Context, fullName, email, passwordHash, universityID string) (*User, error) {
	row, err := r.queries.CreateRepresentativeUser(ctx, sqlc.CreateRepresentativeUserParams{
		FullName: fullName,
		Email:    email,
		Password: &passwordHash,
		RepresentativeUniversityID: uuidFromString(universityID),
	})
	if err != nil {
		return nil, err
	}
	return toDomainUser(row), nil
}

func toDomainUser(u sqlc.User) *User {
	return &User{
		ID:                       u.ID,
		FullName:                 u.FullName,
		Email:                    u.Email,
		Password:                 u.Password,
		Provider:                 u.Provider,
		ProviderID:               u.ProviderID,
		Avatar:                   u.Avatar,
		EmailVerified:            u.EmailVerified,
		Role:                     u.Role,
		RepresentativeUniversityID: uuidToStringPtr(u.RepresentativeUniversityID),
		CreatedAt:                u.CreatedAt,
		UpdatedAt:                u.UpdatedAt,
	}
}

func uuidToStringPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := formatUUID(u.Bytes)
	return &s
}

func uuidFromString(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{}
	}
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}
	}
	return u
}

// formatUUID renders a 16-byte UUID in canonical 8-4-4-4-12 form.
func formatUUID(b [16]byte) string {
	hex := make([]byte, 36)
	const digits = "0123456789abcdef"
	pos := 0
	for i, by := range b {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			hex[pos] = '-'
			pos++
		}
		hex[pos] = digits[by>>4]
		hex[pos+1] = digits[by&0x0f]
		pos += 2
	}
	return string(hex)
}
