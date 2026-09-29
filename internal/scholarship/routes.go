package scholarship

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	r chi.Router,
	h *ScholarshipHandler,
	authMW func(http.Handler) http.Handler,
	adminMW func(http.Handler) http.Handler,
	optionalAuthMW func(http.Handler) http.Handler,
) {
	// Create, update, publish, and delete are admin-only in v1 — scholarships are
	// catalog content the admin curates. (Representative ownership is a planned
	// follow-up; the university_id/college_id FKs are already in place for it.)
	r.With(authMW, adminMW).Post("/api/v1/scholarships", h.Create)
	r.With(authMW, adminMW).Post("/api/v1/scholarships/{id}/publish", h.Publish)
	r.With(authMW, adminMW).Patch("/api/v1/scholarships/{id}", h.Update)
	r.With(authMW, adminMW).Delete("/api/v1/scholarships/{id}", h.Delete)

	// Reference/lookup lists are public so the frontend can build filter forms.
	r.Get("/api/v1/scholarships/education-levels", h.GetEducationLevels)
	r.Get("/api/v1/scholarships/demographics", h.GetDemographics)
	r.Get("/api/v1/scholarships/lookups", h.GetAllLookups)

	// Reads are public; OptionalAuthMiddleware stamps is_favorited when a valid
	// session cookie is present. Non-admin callers only ever see published rows.
	r.With(optionalAuthMW).Get("/api/v1/scholarships/search", h.Search)
	r.With(optionalAuthMW).Get("/api/v1/scholarships", h.Get)
	r.With(optionalAuthMW).Get("/api/v1/scholarships/{id}", h.GetByID)
}
