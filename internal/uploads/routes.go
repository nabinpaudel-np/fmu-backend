package uploads

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	r chi.Router,
	h *UploadsHandler,
	authMW func(http.Handler) http.Handler,
	adminOrRepMW func(http.Handler) http.Handler,
) {
	// /sign and /image are admin OR representative — branding assets the
	// rep needs to refresh on their own university. Binding the URL back
	// to the rep's university happens later at PATCH time via
	// RequireUniversityEditor.
	r.With(authMW, adminOrRepMW).Post("/api/v1/uploads/sign", h.Sign)
	r.With(authMW, adminOrRepMW).Post("/api/v1/uploads/image", h.UploadImage)
	// /document is public so anonymous claim submitters can upload a PDF or
	// image verification proof without first having to register an account.
	r.Post("/api/v1/uploads/document", h.UploadDocument)
	// /resume is public so anonymous counselling submitters can attach a
	// CV without registering. PDF/DOC/DOCX, 5 MB cap.
	r.Post("/api/v1/uploads/resume", h.UploadResume)
}
