package programs

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	r chi.Router,
	h *ProgramHandler,
	authMW func(http.Handler) http.Handler,
	adminMW func(http.Handler) http.Handler,
) {
	// Create, update, and delete are admin-only — programs are catalog content
	// the admin curates. Reads are public so the frontend can render lists
	// without forcing callers to log in.
	r.With(authMW, adminMW).Post("/api/v1/programs", h.Create)
	r.With(authMW, adminMW).Put("/api/v1/programs/{id}", h.Update)
	r.With(authMW, adminMW).Delete("/api/v1/programs/{id}", h.Delete)

	r.Get("/api/v1/programs", h.List)
	r.Get("/api/v1/programs/all", h.ListAll)
	r.Get("/api/v1/programs/{id}", h.GetByID)
}
