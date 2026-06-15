package token

import "time"

type RefreshToken struct {
	ID        string
	Token     string
	UserID    string
	UserAgent *string
	ExpiresAt time.Time
	CreatedAt time.Time
	Revoked   bool
}