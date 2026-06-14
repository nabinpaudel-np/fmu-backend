package user

import "time"

type User struct {
	ID            string
	FullName      string
	Email         string
	Password      *string
	Provider      *string
	ProviderID    *string
	Avatar        *string
	EmailVerified bool
	Role          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
