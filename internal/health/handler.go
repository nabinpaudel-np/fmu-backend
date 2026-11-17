// Package health answers the questions a platform asks about a running
// service: is the process alive, and is it ready to take traffic.
//
// The two are separate on purpose. Liveness must not depend on anything
// that can fail on its own, because a liveness check that fails when the
// database is down makes the platform restart a perfectly healthy process.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"fmu-backend/internal/nlsearch"
)

// Handler serves the health endpoints.
type Handler struct {
	pool *pgxpool.Pool
	ai   AIStatus
}

// AIStatus reports the state of the language model. It is an interface so
// this package does not depend on the search package.
type AIStatus interface {
	// KeyStatuses reports the state of every key.
	KeyStatuses() []nlsearch.KeyStatus
	// Model reports the model in use.
	Model() string
}

// NewHandler builds a handler. A nil pool means the database is not
// checked, which is what a liveness check wants. A nil AI reporter means
// the model is not configured.
func NewHandler(pool *pgxpool.Pool, ai AIStatus) *Handler {
	return &Handler{pool: pool, ai: ai}
}

// RegisterRoutes wires the health endpoints.
//
// None of them is behind the auth middleware: a platform probing the
// service has no session to offer.
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/health", h.Liveness)
	r.Get("/health/ready", h.Readiness)
	r.Get("/health/ai", h.Model)
}

// Liveness says the process is up and serving. It touches nothing that can
// fail on its own, so a failure here means the process itself is broken.
func (h *Handler) Liveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, Status{Status: "ok"})
}

// Readiness says the process can take traffic. The database is checked,
// because a process that cannot reach the database cannot answer a request.
func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	if h.pool == nil {
		writeJSON(w, http.StatusOK, Status{Status: "ok"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
	defer cancel()

	var one int
	if err := h.pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, Status{
			Status: "not ready",
			Error:  "database unreachable: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, Status{Status: "ok"})
}

// Model reports the state of the language model.
//
// The model is the one dependency that can fail while the rest of the API
// keeps working: the day's allowance can run out, a key can be rejected,
// the provider can be busy. This says which of those it is, and which key
// will serve the next search.
//
// It does not call the provider. A check that spent a request from the
// day's allowance would make the thing it is checking worse.
func (h *Handler) Model(w http.ResponseWriter, r *http.Request) {
	if h.ai == nil {
		writeJSON(w, http.StatusServiceUnavailable, AIHealth{
			Status:  "not configured",
			Message: "no language model is configured, natural-language search is disabled",
		})
		return
	}

	keys := h.ai.KeyStatuses()
	available := 0
	for _, key := range keys {
		if key.Available {
			available++
		}
	}

	status := AIHealth{
		Status:        "ok",
		Model:         h.ai.Model(),
		Keys:          keys,
		KeysAvailable: available,
		KeysTotal:     len(keys),
	}

	if available == 0 {
		status.Status = "no key available"
		status.Message = "every key is out of allowance, natural-language search will fail until one recovers"
		writeJSON(w, http.StatusServiceUnavailable, status)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// AIHealth is the body of the model health response.
type AIHealth struct {
	// Status is "ok", "not configured" or "no key available".
	Status string `json:"status" example:"ok"`
	// Model is the model in use.
	Model string `json:"model,omitempty" example:"gemini-3.5-flash-lite"`
	// Keys is the state of every key.
	Keys []nlsearch.KeyStatus `json:"keys,omitempty"`
	// KeysAvailable is how many keys can be used right now.
	KeysAvailable int `json:"keys_available"`
	// KeysTotal is how many keys are configured.
	KeysTotal int `json:"keys_total"`
	// Message explains a status that is not "ok".
	Message string `json:"message,omitempty"`
}

// readinessTimeout is how long the database is given to answer.
const readinessTimeout = 3 * time.Second

// Status is the body of a health response.
type Status struct {
	Status string `json:"status" example:"ok"`
	Error  string `json:"error,omitempty" example:"database unreachable"`
}

// writeJSON writes the body. The health endpoints do not use the shared
// response envelope: a platform reading these wants the status in the body
// itself, and some of them only look at the status code.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// The status code has already been sent by this point, so an encoding
	// error is deliberately ignored.
	_ = json.NewEncoder(w).Encode(body)
}
