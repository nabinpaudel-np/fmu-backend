package passwordreset

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the two public endpoints. Both are unauthenticated
// by design — the token in the reset link *is* the credential, so gating
// it behind an authenticated route would defeat itself.
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Post("/api/v1/auth/forgot-password", h.ForgotPassword)
	r.Post("/api/v1/auth/reset-password", h.ResetPassword)
}
