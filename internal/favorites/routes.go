package favorites

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	r chi.Router,
	h *Handler,
	authMW func(http.Handler) http.Handler,
	studentMW func(http.Handler) http.Handler,
) {
	r.With(authMW, studentMW).Post("/api/v1/favorites/universities/{id}", h.FavoriteUniversity)
	r.With(authMW, studentMW).Delete("/api/v1/favorites/universities/{id}", h.UnfavoriteUniversity)
	r.With(authMW, studentMW).Get("/api/v1/favorites/universities", h.ListUniversities)

	r.With(authMW, studentMW).Post("/api/v1/favorites/colleges/{id}", h.FavoriteCollege)
	r.With(authMW, studentMW).Delete("/api/v1/favorites/colleges/{id}", h.UnfavoriteCollege)
	r.With(authMW, studentMW).Get("/api/v1/favorites/colleges", h.ListColleges)
}