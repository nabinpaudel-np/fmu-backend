package uploads

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	r chi.Router,
	h *UploadsHandler,
	authMW func(http.Handler) http.Handler,
) {
	// /sign and /image accept any authenticated user; the handler enforces
	// the role × purpose matrix (admin/rep → logo/cover/gallery/avatar;
	// student → avatar only). Branding assets are still bound back to a
	// specific university/college at PATCH time via RequireUniversityEditor.
	r.With(authMW).Post("/api/v1/uploads/sign", h.Sign)
	r.With(authMW).Post("/api/v1/uploads/image", h.UploadImage)
	// /document is public so anonymous claim submitters can upload a PDF or
	// image verification proof without first having to register an account.
	r.Post("/api/v1/uploads/document", h.UploadDocument)
	// /resume is public so anonymous counselling submitters can attach a
	// CV without registering. PDF/DOC/DOCX, 5 MB cap.
	r.Post("/api/v1/uploads/resume", h.UploadResume)
}
