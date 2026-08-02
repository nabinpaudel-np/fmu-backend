package passwordreset

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"fmu-backend/internal/db/sqlc"
)

// Repository wraps the sqlc-generated queries for password_reset_tokens
// (creates + reads + used-marking) and shields the service from the
// generated package. Token lookups are by SHA-256 hash, never by id.
type Repository interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*PasswordResetToken, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	MarkUsed(ctx context.Context, id string) error
}

type repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) Repository {
	return &repository{queries: queries}
}

func (r *repository) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*PasswordResetToken, error) {
	row, err := r.queries.CreatePasswordResetToken(ctx, sqlc.CreatePasswordResetTokenParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, err
	}
	return toDomain(row), nil
}

func (r *repository) GetByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error) {
	row, err := r.queries.GetPasswordResetTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if row.ID == "" {
		return nil, nil
	}
	return toDomain(row), nil
}

func (r *repository) MarkUsed(ctx context.Context, id string) error {
	return r.queries.MarkPasswordResetTokenUsed(ctx, id)
}

func toDomain(t sqlc.PasswordResetToken) *PasswordResetToken {
	var usedAt *time.Time
	if t.UsedAt.Valid {
		u := t.UsedAt.Time
		usedAt = &u
	}
	return &PasswordResetToken{
		ID:        t.ID,
		UserID:    t.UserID,
		ExpiresAt: t.ExpiresAt,
		UsedAt:    usedAt,
		CreatedAt: t.CreatedAt,
	}
}
