package claim

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	r chi.Router,
	h *ClaimHandler,
	authMW func(http.Handler) http.Handler,
	adminMW func(http.Handler) http.Handler,
	optionalAuthMW func(http.Handler) http.Handler,
) {

	r.With(optionalAuthMW).Post("/api/v1/claims/universities/{id}", h.Submit(TargetUniversity))
	r.With(optionalAuthMW).Post("/api/v1/claims/colleges/{id}", h.Submit(TargetCollege))

	r.With(authMW, adminMW).Get("/api/v1/admin/claims", h.List)
	r.With(authMW, adminMW).Get("/api/v1/admin/claims/{id}", h.GetByID)
	r.With(authMW, adminMW).Post("/api/v1/admin/claims/{id}/approve", h.Approve)
	r.With(authMW, adminMW).Post("/api/v1/admin/claims/{id}/reject", h.Reject)
}
