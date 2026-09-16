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
	ErrEmailCannotBeChanged               = errors.New("email cannot be changed via this endpoint; contact FMU admin to update your email")
	ErrRepOutOfScope                      = errors.New("representative can only edit their own university")
	ErrRepUniversityIDRequired            = errors.New("representatives must specify a parent university on create")
	ErrRepCannotChangeNameOrSlug          = errors.New("representatives cannot change name or slug")
	ErrProgramDegreeNotFound              = errors.New("degree does not exist")
	ErrPublishMissingFields               = errors.New("missing required fields for publish")
	ErrInvalidPasswordResetToken          = errors.New("invalid password reset token")
	ErrPasswordResetTokenExpired          = errors.New("password reset token has expired")
	ErrPasswordResetTokenUsed             = errors.New("password reset token has already been used")
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

// RowError describes a single problem found while parsing one CSV row during
// a bulk university or college upload. Row is the 1-based row number in the
// file (the header is row 0; a Column == "header" failure uses Row = 0).
// Column is the CSV column name (snake_case as in the file). Value is the
// offending cell text. Message is a human-readable explanation suitable
// for showing back to the admin who uploaded the CSV.
type RowError struct {
	Row     int    `json:"row"`
	Column  string `json:"column"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
}

// BulkValidationError is returned by the bulk-upload service when one or
// more CSV rows fail validation. The handler maps it to a 400 response
// whose body includes the per-row errors so the admin can fix the file
// and resubmit.
type BulkValidationError struct {
	Errors []RowError
}

func (e *BulkValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "bulk upload validation failed"
	}
	return fmt.Sprintf("bulk upload validation failed: %d row error(s) starting at row %d", len(e.Errors), e.Errors[0].Row)
}
