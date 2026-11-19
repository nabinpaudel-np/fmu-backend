package adminuser

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the admin user endpoints. Every one of them is
// behind the admin role check, so only an admin can reach them.
func RegisterRoutes(
	r chi.Router,
	h *Handler,
	authMW func(http.Handler) http.Handler,
	adminMW func(http.Handler) http.Handler,
) {
	r.With(authMW, adminMW).Get("/api/v1/admin/users", h.List)
	r.With(authMW, adminMW).Get("/api/v1/admin/users/{id}", h.GetByID)
	r.With(authMW, adminMW).Patch("/api/v1/admin/users/{id}/role", h.UpdateRole)
	r.With(authMW, adminMW).Post("/api/v1/admin/users/{id}/reset-password", h.ResetPassword)
	r.With(authMW, adminMW).Delete("/api/v1/admin/users/{id}", h.Delete)
}
