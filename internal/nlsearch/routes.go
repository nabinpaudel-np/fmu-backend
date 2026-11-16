package nlsearch

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes wires the natural-language search endpoint.
func RegisterRoutes(
	r chi.Router,
	h *Handler,
	optionalAuthMW func(http.Handler) http.Handler,
	limiter *RateLimiter,
) {
	r.With(optionalAuthMW, limiter.Limit).Post("/api/v1/nl-search", h.Search)
}
