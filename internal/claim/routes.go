package claim

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes wires both the public submit endpoint (no auth required)
// and the admin moderation endpoints.
//
// Public: POST /api/v1/claims/universities/{id}
// Admin:
//   GET    /api/v1/admin/claims
//   GET    /api/v1/admin/claims/{id}
//   POST   /api/v1/admin/claims/{id}/approve
//   POST   /api/v1/admin/claims/{id}/reject
func RegisterRoutes(
	r chi.Router,
	h *ClaimHandler,
	authMW func(http.Handler) http.Handler,
	adminMW func(http.Handler) http.Handler,
	optionalAuthMW func(http.Handler) http.Handler,
) {
	// Public submission. optionalAuthMW parses the JWT if a cookie is
	// present (without rejecting when it's missing) so the service can
	// reject student/admin role claims with 403. Anonymous requests still
	// pass through with no claims in context and proceed normally.
	r.With(optionalAuthMW).Post("/api/v1/claims/universities/{id}", h.Submit)

	// Admin moderation. authMW must come BEFORE adminMW so the JWT is
	// parsed and claims are injected into the request context — RequireRole
	// reads claims from context and would otherwise 401 every request.
	r.With(authMW, adminMW).Get("/api/v1/admin/claims", h.List)
	r.With(authMW, adminMW).Get("/api/v1/admin/claims/{id}", h.GetByID)
	r.With(authMW, adminMW).Post("/api/v1/admin/claims/{id}/approve", h.Approve)
	r.With(authMW, adminMW).Post("/api/v1/admin/claims/{id}/reject", h.Reject)
}
