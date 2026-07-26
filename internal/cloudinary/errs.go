package cloudinary

import "errors"

var (
	// ErrNotConfigured: required env vars are missing or runtime not initialized.
	ErrNotConfigured = errors.New("cloudinary: not configured")
	// ErrUploadFailed: SDK or network error during upload.
	ErrUploadFailed = errors.New("cloudinary: upload failed")
)
