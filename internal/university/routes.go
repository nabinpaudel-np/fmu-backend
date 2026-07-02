package university

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	r chi.Router,
	h *UniversityHandler,
	authMW func(http.Handler) http.Handler,
	adminMW func(http.Handler) http.Handler,
) {
	r.With(authMW, adminMW).Post("/api/v1/universities", h.Create)

	// Static, child-resource routes — must come before /{id} so chi's
	// static-segment precedence resolves them first.
	r.Get("/api/v1/universities/search", h.Search)
	r.Get("/api/v1/universities/majors", h.GetMajors)
	r.Get("/api/v1/universities/degree-levels", h.GetDegreeLevels)
	r.Get("/api/v1/universities/study-formats", h.GetStudyFormats)
	r.Get("/api/v1/universities/special-affiliations", h.GetSpecialAffiliations)
	r.Get("/api/v1/universities/athletics", h.GetAthletics)
	r.Get("/api/v1/universities/support-services", h.GetSupportServices)
	r.Get("/api/v1/universities/lookups", h.GetAllLookups)

	// List + detail — /{id} last so static routes above win.
	r.Get("/api/v1/universities", h.Get)
	r.Get("/api/v1/universities/{id}", h.GetByID)
}
