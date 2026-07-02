package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *AuthHandler, authMW func(http.Handler) http.Handler) {
	r.Post("/api/v1/auth/register", h.Register)
	r.Post("/api/v1/auth/login", h.Login)
	r.Post("/api/v1/auth/refresh", h.Refresh)
	r.Get("/api/v1/auth/google", h.GoogleLogin)
	r.Get("/api/v1/auth/google/callback", h.GoogleCallback)
	r.Delete("/api/v1/auth/logout", h.Logout)

	r.With(authMW).Get("/api/v1/auth/me", h.Me)
}
