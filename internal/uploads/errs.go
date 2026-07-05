package uploads

import "errors"

var (
	// ErrInvalidPurpose: purpose isn't in the allowlist.
	ErrInvalidPurpose = errors.New("uploads: invalid purpose")
	// ErrTooLarge: uploaded file exceeded the configured size limit.
	ErrTooLarge = errors.New("uploads: file too large")
	// ErrUnsupportedMime: uploaded file's MIME type isn't allowlisted.
	ErrUnsupportedMime = errors.New("uploads: unsupported mime type")
	// ErrMissingFile: multipart "file" field was empty/missing.
	ErrMissingFile = errors.New("uploads: file is required")
)
