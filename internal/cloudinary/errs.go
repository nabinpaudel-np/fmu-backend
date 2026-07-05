package cloudinary

import "errors"

var (
	// ErrNotConfigured: required env vars are missing or runtime not initialized.
	ErrNotConfigured = errors.New("cloudinary: not configured")
	// ErrUploadFailed: SDK or network error during upload/destroy.
	ErrUploadFailed = errors.New("cloudinary: upload failed")
	// ErrNotOurAsset: URL is not a Cloudinary delivery URL or has no parseable id.
	ErrNotOurAsset = errors.New("cloudinary: not our asset")
)
