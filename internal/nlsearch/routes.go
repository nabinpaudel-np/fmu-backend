package nlsearch

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes wires the natural-language search endpoint.
//
// This is a separate endpoint from the catalogue's own search routes, which
// keep serving the frontend's structured filters untouched. The optional
// auth middleware lets a signed-in caller be rate-limited by their own
// budget rather than by their address.
func RegisterRoutes(
	r chi.Router,
	h *Handler,
	optionalAuthMW func(http.Handler) http.Handler,
	limiter *RateLimiter,
) {
	r.With(optionalAuthMW, limiter.Limit).Post("/api/v1/nl-search", h.Search)
}
