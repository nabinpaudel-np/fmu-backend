package supabase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type Config struct {
	URL           string
	ServiceRoleKey string
}

type Client struct {
	cfg  Config
	hc   *http.Client
	now  func() time.Time
}

type UploadResult struct {
	Path      string
	PublicURL string
	Bytes     int
}

func New(cfg Config) (*Client, error) {
	if cfg.URL == "" || cfg.ServiceRoleKey == "" {
		return nil, fmt.Errorf("%w: SUPABASE_URL and SUPABASE_SERVICE_ROLE_KEY are required", ErrNotConfigured)
	}
	cfg.URL = strings.TrimRight(cfg.URL, "/")
	return &Client{
		cfg:  cfg,
		hc:   &http.Client{Timeout: 30 * time.Second},
		now:  time.Now,
	}, nil
}

// Upload writes the file to {bucket}/{path}. The bucket must be public for the
// returned PublicURL to be reachable; the call itself works for either, but
// the URL we hand back is the public-v1 form.
func (c *Client) Upload(ctx context.Context, bucket, objectPath, contentType string, file io.Reader) (UploadResult, error) {
	if c.cfg.URL == "" || c.cfg.ServiceRoleKey == "" {
		return UploadResult{}, ErrNotConfigured
	}
	if bucket == "" || objectPath == "" {
		return UploadResult{}, fmt.Errorf("%w: bucket and path are required", ErrNotConfigured)
	}

	body, err := io.ReadAll(file)
	if err != nil {
		return UploadResult{}, fmt.Errorf("%w: read file: %v", ErrUploadFailed, err)
	}

	uploadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s",
		c.cfg.URL, url.PathEscape(bucket), url.PathEscape(objectPath))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(body))
	if err != nil {
		return UploadResult{}, fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.ServiceRoleKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "false")

	resp, err := c.hc.Do(req)
	if err != nil {
		return UploadResult{}, fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return UploadResult{}, fmt.Errorf("%w: HTTP %d: %s", ErrUploadFailed, resp.StatusCode, string(respBody))
	}

	var ack struct {
		Key string `json:"Key"`
	}
	_ = json.Unmarshal(respBody, &ack)

	return UploadResult{
		Path:      path.Join(bucket, objectPath),
		PublicURL: c.PublicURL(bucket, objectPath),
		Bytes:     len(body),
	}, nil
}

// PublicURL returns the public-v1 URL for an object. Reachable only if the
// bucket is marked Public in the Supabase dashboard.
func (c *Client) PublicURL(bucket, objectPath string) string {
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s",
		c.cfg.URL, url.PathEscape(bucket), url.PathEscape(objectPath))
}
