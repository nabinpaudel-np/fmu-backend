package auth

import (
	"context"
	"errors"
	"fmu-backend/internal/token"
)

type ctxkey int

const userClaimsKey ctxkey = iota

var ErrNoUserInContext = errors.New("no user in context")

func WithClaims(ctx context.Context, claims *token.AccessTokenClaims) context.Context {
	return context.WithValue(ctx, userClaimsKey, claims)
}

func ClaimsFromContext(ctx context.Context) (*token.AccessTokenClaims, error) {
	c, ok := ctx.Value(userClaimsKey).(*token.AccessTokenClaims)
	if !ok || c == nil {
		return nil, ErrNoUserInContext
	}
	return c, nil
}

// OptionalUserID returns the authenticated user's id when OptionalAuthMiddleware
// (or AuthMiddleware) injected claims, otherwise ("", false). Use this on
// endpoints that personalize responses without requiring auth.
func OptionalUserID(ctx context.Context) (string, bool) {
	c, ok := ctx.Value(userClaimsKey).(*token.AccessTokenClaims)
	if !ok || c == nil {
		return "", false
	}
	return c.UserID, true
}
