package blog

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	r chi.Router,
	h *BlogHandler,
	authMW func(http.Handler) http.Handler,
	adminMW func(http.Handler) http.Handler,
) {
	// Admin-only: full CRUD, publish toggle, admin list, get-by-id.
	r.With(authMW, adminMW).Get("/api/v1/blogs/all", h.ListAdmin)
	r.With(authMW, adminMW).Get("/api/v1/blogs/id/{id}", h.GetByID)
	r.With(authMW, adminMW).Post("/api/v1/blogs", h.Create)
	r.With(authMW, adminMW).Put("/api/v1/blogs/{id}", h.Update)
	r.With(authMW, adminMW).Patch("/api/v1/blogs/{id}", h.Update)
	r.With(authMW, adminMW).Delete("/api/v1/blogs/{id}", h.Delete)
	r.With(authMW, adminMW).Post("/api/v1/blogs/{id}/publish", h.Publish)

	// Public: published-only reads.
	r.Get("/api/v1/blogs", h.ListPublic)
	// Note: the {slug} route must come AFTER any concrete admin subroutes
	// that share the same prefix. chi resolves routes in registration
	// order, so /all and /id/{id} above win for those exact segments.
	r.Get("/api/v1/blogs/{slug}", h.GetBySlug)
}
