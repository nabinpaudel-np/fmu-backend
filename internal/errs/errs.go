package errs

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInternalServer                     = errors.New("internal server error")
	ErrNotFound                           = errors.New("not found")
	ErrUnauthorized                       = errors.New("unauthorized")
	ErrInvalidCredentials                 = errors.New("invalid credentials")
	ErrInvalidToken                       = errors.New("invalid token")
	ErrUserNotFound                       = errors.New("user not found")
	ErrUserAlreadyExists                  = errors.New("user already exists")
	ErrForbidden                          = errors.New("forbidden")
	ErrBadRequest                         = errors.New("bad request")
	ErrEmailAlreadyRegistered             = errors.New("email already registered with password login")
	ErrRefreshTokenExpired                = errors.New("refresh token has expired")
	ErrRefreshTokenRevoked                = errors.New("refresh token has been revoked")
	ErrInvalidRefreshToken                = errors.New("invalid refresh token")
	ErrUniversitySlugTaken                = errors.New("university with this slug already exists")
	ErrCollegeSlugTaken                   = errors.New("college with this slug already exists")
	ErrCollegeUniversityNotFound          = errors.New("parent university does not exist")
	ErrClaimAlreadyReviewed               = errors.New("claim has already been reviewed")
	ErrUniversityAlreadyHasRepresentative = errors.New("university already has a representative")
	ErrCollegeAlreadyHasRepresentative    = errors.New("college already has a representative")
	ErrClaimRoleNotAllowed                = errors.New("only non-student, non-admin users may submit a claim")
	ErrRepOutOfScope                      = errors.New("representative can only edit their own university")
	ErrRepCannotChangeNameOrSlug          = errors.New("representatives cannot change name or slug")
	ErrProgramDegreeNotFound              = errors.New("degree does not exist")
	ErrPublishMissingFields               = errors.New("missing required fields for publish")
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

// PublishValidationError is returned when a draft university or college is
// published before all required fields are filled in. Fields lists the
// request-DTO field names that are still empty/missing. The handler maps
// it to a 400 with the field list.
type PublishValidationError struct {
	Fields []string
}

func (e *PublishValidationError) Error() string {
	return ErrPublishMissingFields.Error() + ": " + strings.Join(e.Fields, ", ")
}
