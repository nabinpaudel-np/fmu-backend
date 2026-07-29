package college

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"fmu-backend/internal/auth"
)

func RegisterRoutes(
	r chi.Router,
	h *CollegeHandler,
	authMW func(http.Handler) http.Handler,
	adminMW func(http.Handler) http.Handler,
	adminOrRepMW func(http.Handler) http.Handler,
	optionalAuthMW func(http.Handler) http.Handler,
) {
	// Create is allowed for admins (any university) and university-scoped
	// representatives (their own university — verified in the service via
	// claims).
	r.With(authMW, adminOrRepMW).Post("/api/v1/colleges", h.Create)

	// Publish is admin-only — it re-runs required-field validation and
	// flips status to "published". Reps can save drafts but only admins
	// can promote them.
	r.With(authMW, adminMW).Post("/api/v1/colleges/{id}/publish", h.Publish)

	// Update is gated by RequireCollegeEditor: admins always pass,
	// college-scoped representatives only pass when their bound college id
	// matches the URL. University-scoped representatives are rejected — they
	// can Create colleges under their university but cannot Patch individual
	// colleges (different scope).
	r.With(authMW, auth.RequireCollegeEditor("id")).Patch("/api/v1/colleges/{id}", h.Update)

	r.With(optionalAuthMW).Get("/api/v1/colleges", h.List)
	r.With(optionalAuthMW).Get("/api/v1/colleges/search", h.Search)
	r.With(optionalAuthMW).Get("/api/v1/colleges/{id}", h.GetByID)

	r.With(optionalAuthMW).Get("/api/v1/universities/{universityID}/colleges", h.ListByUniversity)
}
