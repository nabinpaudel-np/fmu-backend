package uploads

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	r chi.Router,
	h *UploadsHandler,
	authMW func(http.Handler) http.Handler,
	adminMW func(http.Handler) http.Handler,
) {
	r.With(authMW, adminMW).Post("/api/v1/uploads/sign", h.Sign)
	r.With(authMW, adminMW).Post("/api/v1/uploads/image", h.UploadImage)
}
