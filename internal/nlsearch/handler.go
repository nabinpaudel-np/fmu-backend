package nlsearch

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"fmu-backend/internal/response"
)

// Handler serves the natural-language search endpoint.
type Handler struct {
	svc Service
}

// NewHandler builds the handler.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Search answers a question about the catalogue.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.svc.Search(r.Context(), req)
	if err != nil {
		if !errors.Is(err, ErrEmptyQuery) && !errors.Is(err, ErrQueryTooLong) {
			log.Printf("nl-search: %v", err)
		}
		h.writeError(w, err)
		return
	}
	response.Success(w, http.StatusOK, res)
}

// writeError maps a service error onto a status code.
func (h *Handler) writeError(w http.ResponseWriter, err error) {
	var verr *ValidationError
	switch {
	case errors.Is(err, ErrEmptyQuery):
		response.Error(w, http.StatusBadRequest, "a query is required")
	case errors.Is(err, ErrQueryTooLong):
		response.Error(w, http.StatusBadRequest, "the query is too long")
	case errors.As(err, &verr):
		response.ValidationError(w, http.StatusUnprocessableEntity, []response.ErrorDetail{
			{Field: verr.Field, Message: verr.Message},
		})
	case errors.Is(err, ErrProviderUnavailable):
		response.Error(w, http.StatusServiceUnavailable, "natural-language search is not available right now")
	case errors.Is(err, ErrProviderQuotaExceeded):
		// The question was fine; the server has spent its allowance for the
		// day. Telling the user to rephrase would send them off to try the
		// same question in ten different ways.
		writeRetryAfter(w, err)
		response.Error(w, http.StatusServiceUnavailable, "the language model's daily allowance is spent, please try again later")
	case errors.Is(err, ErrProviderRateLimited):
		// Too many questions at once. This clears on its own, so the client is
		// told how long to wait.
		writeRetryAfter(w, err)
		response.Error(w, http.StatusServiceUnavailable, "the language model is busy, please try again shortly")
	case errors.Is(err, ErrProviderBusy):
		// The provider is saturated. Nothing about the question is wrong.
		writeRetryAfter(w, err)
		response.Error(w, http.StatusServiceUnavailable, "the language model is busy, please try again shortly")
	case errors.Is(err, ErrProviderFailed):
		response.Error(w, http.StatusBadGateway, "the question could not be understood, please try rephrasing it")
	default:
		response.Error(w, http.StatusInternalServerError, "something went wrong")
	}
}

// writeRetryAfter tells the client how long to wait, when the provider said.
func writeRetryAfter(w http.ResponseWriter, err error) {
	var retryable *retryableError
	if !errors.As(err, &retryable) || retryable.retryAfter <= 0 {
		return
	}
	seconds := int(retryable.retryAfter.Round(time.Second) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
}
