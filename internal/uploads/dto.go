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
	CloudName string `json:"cloud_name"`
	APIKey    string `json:"api_key"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
	Folder    string `json:"folder"`
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
