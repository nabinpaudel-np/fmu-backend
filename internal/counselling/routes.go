package counselling

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes wires both the public submission endpoints (no auth) and
// the admin/representative moderation endpoints. The two PATCH endpoints
// share the same handler — the service does scope checks based on the
// caller's claims.
//
// Public:
//
//	POST /api/v1/counselling/general
//	POST /api/v1/counselling/universities/{id}
//	POST /api/v1/counselling/colleges/{id}
//	POST /api/v1/uploads/resume  (registered in the uploads package)
//
// Admin:
//
//	GET    /api/v1/admin/counselling?type=general|university|college&status=pending|reviewed|archived
//	GET    /api/v1/admin/counselling/{id}
//	PATCH  /api/v1/admin/counselling/{id}
//
// Representative (scoped to their institution by the service):
//
//	GET    /api/v1/representative/counselling
//	GET    /api/v1/representative/counselling/{id}
//	PATCH  /api/v1/representative/counselling/{id}
func RegisterRoutes(
	r chi.Router,
	h *CounsellingHandler,
	authMW func(http.Handler) http.Handler,
	adminMW func(http.Handler) http.Handler,
	adminOrRepMW func(http.Handler) http.Handler,
) {
	// Public submit endpoints. No auth — anonymous visitors land directly.
	r.Post("/api/v1/counselling/general", h.SubmitGeneral)
	r.Post("/api/v1/counselling/universities/{id}", h.SubmitUniversity)
	r.Post("/api/v1/counselling/colleges/{id}", h.SubmitCollege)

	// Admin moderation. authMW must come BEFORE adminMW so the JWT is
	// parsed and claims are injected into context — RequireRole reads
	// claims from context and would otherwise 401 every request.
	r.With(authMW, adminMW).Get("/api/v1/admin/counselling", h.List)
	r.With(authMW, adminMW).Get("/api/v1/admin/counselling/{id}", h.GetByID)
	r.With(authMW, adminMW).Patch("/api/v1/admin/counselling/{id}", h.Update)

	// Representative moderation. Same handler; service enforces that the
	// representative only sees/updates rows bound to their institution.
	r.With(authMW, adminOrRepMW).Get("/api/v1/representative/counselling", h.List)
	r.With(authMW, adminOrRepMW).Get("/api/v1/representative/counselling/{id}", h.GetByID)
	r.With(authMW, adminOrRepMW).Patch("/api/v1/representative/counselling/{id}", h.Update)
}
