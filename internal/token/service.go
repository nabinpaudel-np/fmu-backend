package token

import (
	"context"
	"fmu-backend/internal/config"
	"fmu-backend/internal/errs"
	"time"
)

type TokenService interface {
	CreateAccessToken(userID, email, role string) (string, error)
	CreateRefreshToken(ctx context.Context, userID string, userAgent string) (string, error)
	ValidateRefreshToken(ctx context.Context, token string) (string, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
}

type tokenService struct {
	cfg       *config.Config
	tokenRepo TokenRepository
}

func NewTokenService(tokenRepo TokenRepository, cfg *config.Config) TokenService {
	return &tokenService{
		cfg:       cfg,
		tokenRepo: tokenRepo,
	}
}

func (s *tokenService) CreateAccessToken(userID, email, role string) (string, error) {
	return GenerateAccessToken(userID, email, role, s.cfg.AccessTokenSecret, s.cfg.AccessTokenExpiry)
}

func (s *tokenService) CreateRefreshToken(ctx context.Context, userID string, userAgent string) (string, error) {
	token, err := GenerateRefreshToken()
	if err != nil {
		return "", err
	}
	tokenHash := HashRefreshToken(token)
	expiresAt := time.Now().Add(s.cfg.RefreshTokenExpiry)

	err = s.tokenRepo.Create(ctx, userID, tokenHash, userAgent, expiresAt)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *tokenService) ValidateRefreshToken(ctx context.Context, token string) (string, error) {
	tokenHash := HashRefreshToken(token)
	rt, err := s.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return "", err
	}

	if rt == nil {
		return "", errs.ErrInvalidRefreshToken
	}
	if rt.Revoked {
		return "", errs.ErrRefreshTokenRevoked
	}

	if time.Now().After(rt.ExpiresAt) {
		return "", errs.ErrRefreshTokenExpired
	}

	return rt.UserID, nil
}

func (s *tokenService) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	return s.tokenRepo.DeleteByTokenHash(ctx, tokenHash)
}
