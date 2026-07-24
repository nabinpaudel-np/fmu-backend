package counselling

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
	"fmu-backend/internal/response"
	"fmu-backend/internal/validator"
)

type CounsellingHandler struct {
	svc CounsellingService
}

func NewCounsellingHandler(svc CounsellingService) *CounsellingHandler {
	return &CounsellingHandler{svc: svc}
}

// SubmitGeneral handles POST /api/v1/counselling/general. Public — anyone
// can submit a general counselling inquiry.
func (h *CounsellingHandler) SubmitGeneral(w http.ResponseWriter, r *http.Request) {
	var req SubmitGeneralRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validator.Validate.Struct(&req); err != nil {
		fields := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, fields)
		return
	}

	res, err := h.svc.SubmitGeneral(r.Context(), &req)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusCreated, res)
}

// SubmitUniversity handles POST /api/v1/counselling/universities/{id}.
// Public — anyone can submit, but the university must exist.
func (h *CounsellingHandler) SubmitUniversity(w http.ResponseWriter, r *http.Request) {
	universityID := chi.URLParam(r, "id")

	var req SubmitSpecificRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validator.Validate.Struct(&req); err != nil {
		fields := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, fields)
		return
	}

	res, err := h.svc.SubmitUniversity(r.Context(), universityID, &req)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrNotFound):
			response.Error(w, http.StatusNotFound, "university not found")
		default:
			response.Error(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}
	response.Success(w, http.StatusCreated, res)
}

// SubmitCollege handles POST /api/v1/counselling/colleges/{id}. Public.
func (h *CounsellingHandler) SubmitCollege(w http.ResponseWriter, r *http.Request) {
	collegeID := chi.URLParam(r, "id")

	var req SubmitSpecificRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validator.Validate.Struct(&req); err != nil {
		fields := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, fields)
		return
	}

	res, err := h.svc.SubmitCollege(r.Context(), collegeID, &req)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrNotFound):
			response.Error(w, http.StatusNotFound, "college not found")
		default:
			response.Error(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}
	response.Success(w, http.StatusCreated, res)
}

// List handles GET /api/v1/admin/counselling and /api/v1/representative/counselling.
// Admins see everything (filtered by ?type= and ?status=). Representatives
// are silently scoped to their bound institution (the service enforces this;
// the URL doesn't carry any scope hint).
func (h *CounsellingHandler) List(w http.ResponseWriter, r *http.Request) {
	q := pagination.Parse(r)
	statusFilter := r.URL.Query().Get("status")
	target := TargetType(r.URL.Query().Get("type"))
	if target != "" && !target.IsValid() {
		response.Error(w, http.StatusBadRequest, "type must be one of: general, university, college")
		return
	}

	items, total, err := h.svc.List(r.Context(), target, statusFilter, q)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrBadRequest):
			response.Error(w, http.StatusBadRequest, "status must be one of: pending, reviewed, archived")
		case errors.Is(err, errs.ErrUnauthorized):
			response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		case errors.Is(err, errs.ErrForbidden):
			response.Error(w, http.StatusForbidden, errs.ErrForbidden.Error())
		default:
			response.Error(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	response.Success(w, http.StatusOK, pagination.Response[CounsellingListItem]{
		Items: items,
		Meta:  q.BuildMeta(total),
	})
}

// GetByID handles GET /api/v1/admin/counselling/{id} and /representative/...
func (h *CounsellingHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	item, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrNotFound):
			response.Error(w, http.StatusNotFound, "inquiry not found")
		case errors.Is(err, errs.ErrForbidden):
			response.Error(w, http.StatusForbidden, errs.ErrForbidden.Error())
		default:
			response.Error(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}
	response.Success(w, http.StatusOK, item)
}

// Update handles PATCH .../counselling/{id}. Admin or representative; the
// service enforces representative scope and 403s on out-of-scope rows.
func (h *CounsellingHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validator.Validate.Struct(&req); err != nil {
		fields := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, fields)
		return
	}

	// Defensive: if claims are missing entirely the auth middleware missed
	// something. Fail closed rather than stamping reviewer_id = NULL.
	if _, err := auth.ClaimsFromContext(r.Context()); err != nil {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}

	item, err := h.svc.Update(r.Context(), id, &req)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrNotFound):
			response.Error(w, http.StatusNotFound, "inquiry not found")
		case errors.Is(err, errs.ErrForbidden):
			response.Error(w, http.StatusForbidden, errs.ErrForbidden.Error())
		default:
			response.Error(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}
	response.Success(w, http.StatusOK, item)
}
