package uploads

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"fmu-backend/internal/cloudinary"
	"fmu-backend/internal/response"
	"fmu-backend/internal/supabase"
	"fmu-backend/internal/validator"
)

var allowedImageMimes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
	"image/gif":  {},
}

var allowedDocumentMimes = map[string]struct{}{
	"application/pdf": {},
}

const fileFieldName = "file"

type UploadsHandler struct {
	svc UploadsService
}

func NewHandler(svc UploadsService) *UploadsHandler {
	return &UploadsHandler{svc: svc}
}

// sign handles POST /api/v1/uploads/sign — returns Cloudinary signature params so the browser can upload directly to api.cloudinary.com.
func (h *UploadsHandler) Sign(w http.ResponseWriter, r *http.Request) {
	var req SignUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validator.Validate.Struct(&req); err != nil {
		fields := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, fields)
		return
	}

	payload, err := h.svc.Sign(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidPurpose) {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, payload)
}

func (h *UploadsHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	purpose := r.URL.Query().Get("purpose")
	if !isImagePurpose(purpose) {
		response.Error(w, http.StatusBadRequest, "query parameter 'purpose' must be one of: logo, cover, gallery")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxImageBytes)

	if err := r.ParseMultipartForm(MaxImageBytes); err != nil {

		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			response.Error(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("file exceeds %d bytes", MaxImageBytes))
			return
		}
		response.Error(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile(fileFieldName)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "file is required (multipart field 'file')")
		return
	}
	defer file.Close()

	if header.Size <= 0 {
		response.Error(w, http.StatusBadRequest, "file is empty")
		return
	}

	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	mime := http.DetectContentType(head[:n])
	if _, ok := allowedImageMimes[mime]; !ok {
		response.Error(w, http.StatusUnsupportedMediaType,
			fmt.Sprintf("mime type %s is not allowed", mime))
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	out, err := h.svc.UploadImage(r.Context(), purpose, file)
	if err != nil {
		if errors.Is(err, cloudinary.ErrUploadFailed) || errors.Is(err, cloudinary.ErrNotConfigured) {
			response.Error(w, http.StatusBadGateway, "upload to storage failed")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusCreated, out)
}

// UploadDocument handles POST /api/v1/uploads/document?purpose=document. Only
// PDFs are accepted — these are the verification documents attached to a
// university claim. Uploaded as a Cloudinary "raw" resource so it can be
// downloaded without any image transformations.
func (h *UploadsHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	purpose := r.URL.Query().Get("purpose")
	if !strings.EqualFold(purpose, "document") {
		response.Error(w, http.StatusBadRequest, "query parameter 'purpose' must be 'document'")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxDocumentBytes)

	if err := r.ParseMultipartForm(MaxDocumentBytes); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			response.Error(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("file exceeds %d bytes", MaxDocumentBytes))
			return
		}
		response.Error(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile(fileFieldName)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "file is required (multipart field 'file')")
		return
	}
	defer file.Close()

	if header.Size <= 0 {
		response.Error(w, http.StatusBadRequest, "file is empty")
		return
	}

	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	mime := http.DetectContentType(head[:n])
	if _, ok := allowedDocumentMimes[mime]; !ok {
		response.Error(w, http.StatusUnsupportedMediaType,
			fmt.Sprintf("mime type %s is not allowed (only application/pdf)", mime))
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	out, err := h.svc.UploadDocument(r.Context(), purpose, file, mime)
	if err != nil {
		if errors.Is(err, supabase.ErrUploadFailed) || errors.Is(err, supabase.ErrNotConfigured) {
			response.Error(w, http.StatusBadGateway, "upload to storage failed")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusCreated, out)
}

func isImagePurpose(p string) bool {
	switch p {
	case "logo", "cover", "gallery":
		return true
	}
	return false
}
