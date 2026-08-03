package passwordreset

import "time"

// Domain type mirroring sqlc.PasswordResetToken. We keep an id+hash+expiry
// shape here so callers don't have to import the generated sqlc package.
type PasswordResetToken struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
