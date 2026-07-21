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
	// /document is fully public so anonymous claim submitters can upload
	// their verification PDF without first having to register an account.
	// The 20MB cap + Cloudinary quota are the abuse floor.
	r.Post("/api/v1/uploads/document", h.UploadDocument)
}
