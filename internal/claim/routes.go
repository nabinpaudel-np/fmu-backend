package claim

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes wires both the public submit endpoints (no auth required)
// and the admin moderation endpoints. The submit handler is closed over the
// target type at registration time, so the same ClaimHandler serves both
// /claims/universities/{id} and /claims/colleges/{id} without the route
// needing to parse the type.
//
// Public: POST /api/v1/claims/universities/{id}
//
//	POST /api/v1/claims/colleges/{id}
//
// Admin:
//
//	GET    /api/v1/admin/claims?type=university|college&status=pending|approved|rejected
//	GET    /api/v1/admin/claims/{id}
//	POST   /api/v1/admin/claims/{id}/approve
//	POST   /api/v1/admin/claims/{id}/reject
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
	r.With(optionalAuthMW).Post("/api/v1/claims/universities/{id}", h.Submit(TargetUniversity))
	r.With(optionalAuthMW).Post("/api/v1/claims/colleges/{id}", h.Submit(TargetCollege))

	// Admin moderation. authMW must come BEFORE adminMW so the JWT is
	// parsed and claims are injected into the request context — RequireRole
	// reads claims from context and would otherwise 401 every request.
	r.With(authMW, adminMW).Get("/api/v1/admin/claims", h.List)
	r.With(authMW, adminMW).Get("/api/v1/admin/claims/{id}", h.GetByID)
	r.With(authMW, adminMW).Post("/api/v1/admin/claims/{id}/approve", h.Approve)
	r.With(authMW, adminMW).Post("/api/v1/admin/claims/{id}/reject", h.Reject)
}
