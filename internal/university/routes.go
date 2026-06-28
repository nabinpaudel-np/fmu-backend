package university

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *UniversityHandler) {
	r.Post("/api/v1/universities", h.Create)
	r.Get("/api/v1/universities/majors", h.GetMajors)
	r.Get("/api/v1/universities/degree-levels", h.GetDegreeLevels)
	r.Get("/api/v1/universities/study-formats", h.GetStudyFormats)
	r.Get("/api/v1/universities/special-affiliations", h.GetSpecialAffiliations)
	r.Get("/api/v1/universities/athletics", h.GetAthletics)
	r.Get("/api/v1/universities/support-services", h.GetSupportServices)
	r.Get("/api/v1/universities/lookups", h.GetAllLookups)
}
