package supabase

import "errors"

var (
	// ErrNotConfigured: required env vars are missing or runtime not initialized.
	ErrNotConfigured = errors.New("supabase: not configured")
	// ErrUploadFailed: HTTP or network error during upload.
	ErrUploadFailed = errors.New("supabase: upload failed")
)
