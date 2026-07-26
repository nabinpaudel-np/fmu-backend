package uploads

// allowed purpose values for /sign — frontend passes one of these to
var AllowedPurposes = map[string]struct{}{
	"logo":    {},
	"cover":   {},
	"gallery": {},
}

type SignUploadRequest struct {
	Purpose string `json:"purpose" validate:"required,oneof=logo cover gallery"`
}

type SignUploadResponse struct {
	CloudName    string `json:"cloud_name"`
	APIKey       string `json:"api_key"`
	Timestamp    int64  `json:"timestamp"`
	Signature    string `json:"signature"`
	Folder       string `json:"folder"`
	ResourceType string `json:"resource_type"`
}

type UploadImageResponse struct {
	SecureURL string `json:"secure_url"`
	URL       string `json:"url"`
	PublicID  string `json:"public_id"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Format    string `json:"format,omitempty"`
	Bytes     int    `json:"bytes"`
}

// UploadDocumentResponse is returned by POST /api/v1/uploads/document. Keeps
// the `secure_url` field name so frontend code that already reads it doesn't
// need to change; Path is the Supabase storage key, useful if the bucket is
// ever flipped to private and admins need to mint signed URLs.
type UploadDocumentResponse struct {
	SecureURL string `json:"secure_url"`
	Path      string `json:"path"`
	Bytes     int    `json:"bytes"`
}
