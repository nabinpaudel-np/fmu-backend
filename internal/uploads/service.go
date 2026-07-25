package uploads

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"fmu-backend/internal/cloudinary"
	"fmu-backend/internal/supabase"
)

const (
	MaxImageBytes    int64 = 10 << 20 // 10 MiB
	MaxDocumentBytes int64 = 20 << 20 // 20 MiB
	MaxResumeBytes   int64 = 5 << 20  // 5 MiB
)

type UploadsService interface {
	Sign(ctx context.Context, req SignUploadRequest) (SignUploadResponse, error)
	UploadImage(ctx context.Context, purpose string, file io.Reader) (UploadImageResponse, error)
	UploadDocument(ctx context.Context, purpose string, file io.Reader, mime string) (UploadDocumentResponse, error)
	UploadResume(ctx context.Context, ext string, file io.Reader, mime string) (UploadDocumentResponse, error)
}

type uploadsService struct {
	cld        *cloudinary.Client
	supa       *supabase.Client
	docsBucket string
}

func NewService(c *cloudinary.Client, s *supabase.Client, docsBucket string) UploadsService {
	return &uploadsService{cld: c, supa: s, docsBucket: docsBucket}
}

func (s *uploadsService) Sign(ctx context.Context, req SignUploadRequest) (SignUploadResponse, error) {
	if _, ok := AllowedPurposes[req.Purpose]; !ok {
		return SignUploadResponse{}, ErrInvalidPurpose
	}
	folder, err := s.cld.ResolveFolder(req.Purpose)
	if err != nil {
		return SignUploadResponse{}, err
	}
	payload, err := s.cld.SignUpload(folder, "image")
	if err != nil {
		return SignUploadResponse{}, err
	}
	return SignUploadResponse{
		CloudName:    payload.CloudName,
		APIKey:       payload.APIKey,
		Timestamp:    payload.Timestamp,
		Signature:    payload.Signature,
		Folder:       payload.Folder,
		ResourceType: payload.ResourceType,
	}, nil
}

func (s *uploadsService) UploadImage(ctx context.Context, purpose string, file io.Reader) (UploadImageResponse, error) {
	if _, ok := AllowedPurposes[purpose]; !ok {
		return UploadImageResponse{}, ErrInvalidPurpose
	}
	folder, err := s.cld.ResolveFolder(purpose)
	if err != nil {
		return UploadImageResponse{}, err
	}
	res, err := s.cld.Upload(ctx, folder, file)
	if err != nil {
		return UploadImageResponse{}, err
	}
	return UploadImageResponse{
		SecureURL: res.SecureURL,
		URL:       res.URL,
		PublicID:  res.PublicID,
		Width:     res.Width,
		Height:    res.Height,
		Format:    res.Format,
		Bytes:     res.Bytes,
	}, nil
}

func (s *uploadsService) UploadDocument(ctx context.Context, purpose string, file io.Reader, mime string) (UploadDocumentResponse, error) {
	if purpose != "document" {
		return UploadDocumentResponse{}, ErrInvalidPurpose
	}
	if s.supa == nil {
		return UploadDocumentResponse{}, supabase.ErrNotConfigured
	}

	ext, ok := documentExtensionForMime(mime)
	if !ok {
		return UploadDocumentResponse{}, ErrUnsupportedMime
	}

	name, err := randomObjectName(ext)
	if err != nil {
		return UploadDocumentResponse{}, fmt.Errorf("uploads: %w", err)
	}

	res, err := s.supa.Upload(ctx, s.docsBucket, name, mime, file)
	if err != nil {
		return UploadDocumentResponse{}, err
	}
	return UploadDocumentResponse{
		SecureURL: res.PublicURL,
		Path:      res.Path,
		Bytes:     res.Bytes,
	}, nil
}

func documentExtensionForMime(mime string) (string, bool) {
	switch mime {
	case "application/pdf":
		return "pdf", true
	case "image/jpeg":
		return "jpg", true
	case "image/png":
		return "png", true
	case "image/webp":
		return "webp", true
	case "image/gif":
		return "gif", true
	default:
		return "", false
	}
}

// UploadResume stores a CV/resume in the same Supabase documents bucket but
// accepts PDF/DOC/DOCX with a tighter 5 MB cap. The frontend uploads here,
// captures secure_url, and submits that URL on the counselling form.
func (s *uploadsService) UploadResume(ctx context.Context, ext string, file io.Reader, mime string) (UploadDocumentResponse, error) {
	if s.supa == nil {
		return UploadDocumentResponse{}, supabase.ErrNotConfigured
	}

	name, err := randomObjectName(ext)
	if err != nil {
		return UploadDocumentResponse{}, fmt.Errorf("uploads: %w", err)
	}

	res, err := s.supa.Upload(ctx, s.docsBucket, name, mime, file)
	if err != nil {
		return UploadDocumentResponse{}, err
	}
	return UploadDocumentResponse{
		SecureURL: res.PublicURL,
		Path:      res.Path,
		Bytes:     res.Bytes,
	}, nil
}

func randomObjectName(ext string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + "." + ext, nil
}
