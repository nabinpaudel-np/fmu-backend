package college

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	r chi.Router,
	h *CollegeHandler,
	authMW func(http.Handler) http.Handler,
	adminOrRepMW func(http.Handler) http.Handler,
	optionalAuthMW func(http.Handler) http.Handler,
) {
	// Create is allowed for admins (any university) and representatives
	// (their own university — verified in the service via claims).
	r.With(authMW, adminOrRepMW).Post("/api/v1/colleges", h.Create)

	r.With(optionalAuthMW).Get("/api/v1/colleges", h.List)
	r.With(optionalAuthMW).Get("/api/v1/colleges/search", h.Search)
	r.Get("/api/v1/colleges/{id}", h.GetByID)

	r.With(optionalAuthMW).Get("/api/v1/universities/{universityID}/colleges", h.ListByUniversity)
}