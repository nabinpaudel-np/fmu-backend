package cloudinary

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	cld "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type Config struct {
	CloudName      string
	APIKey         string
	APISecret      string
	Folder         string
	AppEnv         string
	SecureDelivery bool
}

type Client struct {
	cld *cld.Cloudinary
	cfg Config
	now func() time.Time
}

type SignPayload struct {
	CloudName string `json:"cloud_name"`
	APIKey    string `json:"api_key"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
	Folder    string `json:"folder"`
}

type UploadResult struct {
	SecureURL    string `json:"secure_url"`
	URL          string `json:"url"`
	PublicID     string `json:"public_id"`
	AssetFolder  string `json:"asset_folder"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	Format       string `json:"format"`
	Bytes        int    `json:"bytes"`
	ResourceType string `json:"resource_type"`
}

const uploadResourceType = "image"

func New(cfg Config) (*Client, error) {
	if cfg.CloudName == "" || cfg.APIKey == "" || cfg.APISecret == "" {
		if cfg.AppEnv != "development" {
			return nil, fmt.Errorf("%w: CLOUDINARY_CLOUD_NAME / API_KEY / API_SECRET are required (env=%s)", ErrNotConfigured, cfg.AppEnv)
		}
	}
	if cfg.Folder == "" {
		cfg.Folder = "fmu"
	}
	if !cfg.SecureDelivery {
		cfg.SecureDelivery = true
	}

	c, err := cld.NewFromParams(cfg.CloudName, cfg.APIKey, cfg.APISecret)
	if err != nil {
		return nil, fmt.Errorf("cloudinary init: %w", err)
	}
	return &Client{cld: c, cfg: cfg, now: time.Now}, nil
}

func (c *Client) ResolveFolder(purpose string) (string, error) {
	if purpose == "" {
		return "", fmt.Errorf("%w: purpose is required", ErrNotConfigured)
	}
	if strings.ContainsAny(purpose, "/\\") {
		return "", fmt.Errorf("%w: purpose must not contain '/'", ErrNotConfigured)
	}
	return path.Join(c.cfg.Folder, c.cfg.AppEnv, purpose), nil
}

func (c *Client) SignUpload(folder string) (SignPayload, error) {
	if c.cld == nil {
		return SignPayload{}, ErrNotConfigured
	}
	if folder == "" {
		return SignPayload{}, fmt.Errorf("%w: folder is required", ErrNotConfigured)
	}
	ts := c.now().Unix()

	params := url.Values{}
	params.Set("folder", folder)
	params.Set("timestamp", strconv.FormatInt(ts, 10))

	signature, err := signParams(params, c.cfg.APISecret)
	if err != nil {
		return SignPayload{}, fmt.Errorf("cloudinary sign: %w", err)
	}

	return SignPayload{
		CloudName: c.cfg.CloudName,
		APIKey:    c.cfg.APIKey,
		Timestamp: ts,
		Signature: signature,
		Folder:    folder,
	}, nil
}

func (c *Client) Upload(ctx context.Context, folder string, file io.Reader) (UploadResult, error) {
	if c.cld == nil {
		return UploadResult{}, ErrNotConfigured
	}
	if folder == "" {
		return UploadResult{}, fmt.Errorf("%w: folder is required", ErrNotConfigured)
	}

	falseVal := false
	res, err := c.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder:         folder,
		Overwrite:      &falseVal,
		UniqueFilename: &falseVal,
		ResourceType:   uploadResourceType,
	})
	if err != nil {
		return UploadResult{}, fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}

	return UploadResult{
		SecureURL:    res.SecureURL,
		URL:          res.URL,
		PublicID:     res.PublicID,
		AssetFolder:  res.AssetFolder,
		Width:        res.Width,
		Height:       res.Height,
		Format:       res.Format,
		Bytes:        res.Bytes,
		ResourceType: res.ResourceType,
	}, nil
}

func (c *Client) Destroy(ctx context.Context, publicID string) error {
	if c.cld == nil {
		return ErrNotConfigured
	}
	if publicID == "" {
		return fmt.Errorf("%w: public_id is required", ErrNotConfigured)
	}
	_, err := c.cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID:     publicID,
		ResourceType: uploadResourceType,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}
	return nil
}

var secureURLRe = regexp.MustCompile(
	`^https://res\.cloudinary\.com/[^/]+/(?:image|video|raw)/upload/(?:v\d+/)?(.+?)(?:\.[a-zA-Z0-9]+)?$`,
)

func ParsePublicID(secureURL string) (string, error) {
	if secureURL == "" {
		return "", ErrNotOurAsset
	}
	m := secureURLRe.FindStringSubmatch(secureURL)
	if m == nil {
		return "", ErrNotOurAsset
	}
	return m[1], nil
}

func signParams(params url.Values, secret string) (string, error) {
	if secret == "" {
		return "", ErrNotConfigured
	}

	const arrayKeyPattern = `(.*)\[\d+\]`
	ignored := map[string]struct{}{
		"file": {}, "cloud_name": {}, "resource_type": {}, "api_key": {},
	}
	arrayKey := regexp.MustCompile(arrayKeyPattern)

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	cleaned := make(url.Values)
	for _, k := range keys {
		switch {
		case arrayKey.MatchString(k):
			name := arrayKey.FindStringSubmatch(k)[1]
			cleaned[name] = append(cleaned[name], params.Get(k))
		case ignoredKey(ignored, k):
			// skip
		default:
			cleaned[k] = []string{params.Get(k)}
		}
	}

	for k, v := range cleaned {
		cleaned[k] = []string{strings.Join(v, ",")}
	}

	sig, err := api.SignParameters(cleaned, secret)
	if err != nil {
		return "", err
	}
	return sig, nil
}

func ignoredKey(ignored map[string]struct{}, k string) bool {
	_, ok := ignored[k]
	return ok
}
