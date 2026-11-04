package nlsearch

import (
	"encoding/json"
	"errors"
	"net/http"

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
		h.writeError(w, err)
		return
	}
	response.Success(w, http.StatusOK, res)
}

// writeError maps a service error onto a status code. The client is told
// what went wrong only where that is useful and safe: a filter the model got
// wrong is the caller's business, a model failure is not.
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
	case errors.Is(err, ErrProviderFailed):
		response.Error(w, http.StatusBadGateway, "the question could not be understood, please try rephrasing it")
	default:
		response.Error(w, http.StatusInternalServerError, "something went wrong")
	}
}
