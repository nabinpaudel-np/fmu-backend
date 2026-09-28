package studentprofile

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the student profile + recommendations endpoints.
// All routes require an authenticated student — admin and representative
// tokens are rejected by studentMW so they can't read or write student
// preferences (see RequireRole in internal/auth/middleware.go).
func RegisterRoutes(
	r chi.Router,
	h *Handler,
	authMW func(http.Handler) http.Handler,
	studentMW func(http.Handler) http.Handler,
) {
	r.With(authMW, studentMW).Put("/api/v1/student/profile", h.UpdateProfile)
	r.With(authMW, studentMW).Get("/api/v1/student/profile", h.GetProfile)
	r.With(authMW, studentMW).Delete("/api/v1/student/profile", h.DeleteProfile)
	r.With(authMW, studentMW).Get("/api/v1/student/recommendations", h.ListRecommendations)
}