package uploads

import (
	"context"
	"io"

	"fmu-backend/internal/cloudinary"
)

const (
	MaxImageBytes int64 = 10 << 20 // 10 MiB
)

type UploadsService interface {
	Sign(ctx context.Context, req SignUploadRequest) (SignUploadResponse, error)
	UploadImage(ctx context.Context, purpose string, file io.Reader) (UploadImageResponse, error)
}

type uploadsService struct {
	cld *cloudinary.Client
}

func NewService(c *cloudinary.Client) UploadsService {
	return &uploadsService{cld: c}
}

func (s *uploadsService) Sign(ctx context.Context, req SignUploadRequest) (SignUploadResponse, error) {
	if _, ok := AllowedPurposes[req.Purpose]; !ok {
		return SignUploadResponse{}, ErrInvalidPurpose
	}
	folder, err := s.cld.ResolveFolder(req.Purpose)
	if err != nil {
		return SignUploadResponse{}, err
	}
	payload, err := s.cld.SignUpload(folder)
	if err != nil {
		return SignUploadResponse{}, err
	}
	return SignUploadResponse{
		CloudName: payload.CloudName,
		APIKey:    payload.APIKey,
		Timestamp: payload.Timestamp,
		Signature: payload.Signature,
		Folder:    payload.Folder,
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
