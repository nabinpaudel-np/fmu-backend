package counselling

import (
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// stringPtrOrNil trims whitespace and returns nil for empty inputs. Empty
// strings get persisted as NULL rather than "" so JSON responses stay clean.
func stringPtrOrNil(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// uuidFromString parses a textual UUID into a pgtype.UUID. Returns an
// invalid UUID on failure so callers can map the error to 404.
func uuidFromString(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}, err
	}
	return u, nil
}
