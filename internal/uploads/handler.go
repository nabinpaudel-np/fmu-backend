package uploads

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"fmu-backend/internal/auth"
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
	"image/jpeg":      {},
	"image/png":       {},
	"image/webp":      {},
	"image/gif":       {},
}

func isImagePurpose(p string) bool {
	switch p {
	case "logo", "cover", "gallery", "avatar":
		return true
	}
	return false
}

// purposeAllowedForRole decides which roles may upload for a given purpose.
// Branding assets (logo/cover/gallery) belong to a university/college and
// only admins or the bound representative may upload them. Avatars are
// personal — any authenticated user (admin, rep, student) may upload their
// own. Without this check, a student could mint signed Cloudinary uploads
// into the `logo` folder without owning a university.
func purposeAllowedForRole(role, purpose string) bool {
	switch purpose {
	case "logo", "cover", "gallery":
		return role == auth.RoleAdmin || role == auth.RoleRepresentative
	case "avatar":
		return role == auth.RoleAdmin || role == auth.RoleRepresentative || role == auth.RoleStudent
	}
	return false
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

	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil || !purposeAllowedForRole(claims.Role, req.Purpose) {
		response.Error(w, http.StatusForbidden, "your role may not upload for this purpose")
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
		response.Error(w, http.StatusBadRequest, "query parameter 'purpose' must be one of: logo, cover, gallery, avatar")
		return
	}

	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil || !purposeAllowedForRole(claims.Role, purpose) {
		response.Error(w, http.StatusForbidden, "your role may not upload for this purpose")
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

// UploadDocument handles POST /api/v1/uploads/document?purpose=document. PDFs
// and common image formats are stored in Supabase as claim verification proofs.
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
			fmt.Sprintf("mime type %s is not allowed (only PDF and image files)", mime))
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

// UploadResume handles POST /api/v1/uploads/resume. Public — anonymous
// counselling submitters can upload a CV without first registering an account.
// Accepts PDF, DOC, and DOCX (5 MB cap). http.DetectContentType misclassifies
// DOC and DOCX, so we sniff by filename extension. The filename's extension
// is what we trust for storage; MIME is derived from that extension.
func (h *UploadsHandler) UploadResume(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxResumeBytes)

	if err := r.ParseMultipartForm(MaxResumeBytes); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			response.Error(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("file exceeds %d bytes", MaxResumeBytes))
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

	ext, mime, ok := sniffResume(header.Filename)
	if !ok {
		response.Error(w, http.StatusUnsupportedMediaType,
			"only PDF, DOC, and DOCX files are accepted")
		return
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	out, err := h.svc.UploadResume(r.Context(), ext, file, mime)
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

// sniffResume picks the file extension + MIME type from the filename.
// Returns ("", "", false) for anything else.
//
//	Magic bytes (informational only — we don't sniff the body):
//	  PDF  — `%PDF-`
//	  DOCX — `PK\x03\x04` (ZIP/OOXML container)
//	  DOC  — `\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1` (OLE Compound File)
//
// We trust the extension because the frontend always provides the original
// filename via the multipart "file" header.
func sniffResume(filename string) (ext, mime string, ok bool) {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".pdf"):
		return "pdf", "application/pdf", true
	case strings.HasSuffix(lower, ".docx"):
		return "docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", true
	case strings.HasSuffix(lower, ".doc"):
		return "doc", "application/msword", true
	}
	return "", "", false
}
