package university

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"fmu-backend/internal/auth"
)

func RegisterRoutes(
	r chi.Router,
	h *UniversityHandler,
	authMW func(http.Handler) http.Handler,
	adminMW func(http.Handler) http.Handler,
	optionalAuthMW func(http.Handler) http.Handler,
) {
	r.With(authMW, adminMW).Post("/api/v1/universities", h.Create)
	// PATCH is allowed for admins OR the representative assigned to that
	// specific university. RequireUniversityEditor enforces both checks.
	r.With(authMW, auth.RequireUniversityEditor("id")).Patch("/api/v1/universities/{id}", h.Patch)

	r.With(optionalAuthMW).Get("/api/v1/universities/search", h.Search)
	r.With(authMW, adminMW).Get("/api/v1/universities/stats", h.Stats)
	r.Get("/api/v1/universities/majors", h.GetMajors)
	r.Get("/api/v1/universities/degree-levels", h.GetDegreeLevels)
	r.Get("/api/v1/universities/study-formats", h.GetStudyFormats)
	r.Get("/api/v1/universities/special-affiliations", h.GetSpecialAffiliations)
	r.Get("/api/v1/universities/athletics", h.GetAthletics)
	r.Get("/api/v1/universities/support-services", h.GetSupportServices)
	r.Get("/api/v1/universities/lookups", h.GetAllLookups)

	r.With(optionalAuthMW).Get("/api/v1/universities", h.Get)
	r.Get("/api/v1/universities/{id}", h.GetByID)
}
