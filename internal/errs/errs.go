package errs

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInternalServer         = errors.New("internal server error")
	ErrNotFound               = errors.New("not found")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrInvalidToken           = errors.New("invalid token")
	ErrUserNotFound           = errors.New("user not found")
	ErrUserAlreadyExists      = errors.New("user already exists")
	ErrForbidden              = errors.New("forbidden")
	ErrEmailAlreadyRegistered = errors.New("email already registered with password login")
	ErrRefreshTokenExpired    = errors.New("refresh token has expired")
	ErrRefreshTokenRevoked    = errors.New("refresh token has been revoked")
	ErrInvalidRefreshToken    = errors.New("invalid refresh token")
	ErrUniversitySlugTaken    = errors.New("university with this slug already exists")
)

// InvalidReferencesError is returned when one or more UUIDs in a request
// do not exist in the referenced lookup table. Keys are table names
// (e.g. "majors"), values are the missing IDs.
type InvalidReferencesError struct {
	References map[string][]string
}

func (e *InvalidReferencesError) Error() string {
	parts := make([]string, 0, len(e.References))
	for resource, ids := range e.References {
		parts = append(parts, fmt.Sprintf("%s not found: %v", resource, ids))
	}
	return "invalid references: " + strings.Join(parts, "; ")
}
