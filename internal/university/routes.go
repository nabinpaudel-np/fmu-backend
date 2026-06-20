package university

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *UniversityHandler) {
	r.Post("/api/v1/universities", h.Create)

}
