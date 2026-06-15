package token

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenRepository interface {
	Create(ctx context.Context, userID, tokenHash, userAgent string, expiresAt time.Time) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
}

type tokenRepository struct {
	db *pgxpool.Pool
}

func NewTokenRepository(db *pgxpool.Pool) TokenRepository {
	return &tokenRepository{
		db: db,
	}
}

func (r *tokenRepository) Create(ctx context.Context, userID, tokenHash, userAgent string, expiresAt time.Time) error {
	query := `
	INSERT INTO refresh_tokens (user_id, token, user_agent, expires_at)
	VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.Exec(ctx, query, userID, tokenHash, userAgent, expiresAt)

	return err
}

func (r *tokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	query := `
	SELECT id, token, user_id, user_agent, expires_at, created_at, revoked
	FROM refresh_tokens
	WHERE token = $1
	`

	rt := &RefreshToken{}
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&rt.ID,
		&rt.Token,
		&rt.UserID,
		&rt.UserAgent,
		&rt.ExpiresAt,
		&rt.CreatedAt,
		&rt.Revoked,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return rt, err
}

func (r *tokenRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	query := `
	DELETE FROM refresh_tokens
	WHERE token = $1
	`
	_, err := r.db.Exec(ctx, query, tokenHash)
	return err
}
