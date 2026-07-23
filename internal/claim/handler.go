package claim

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

type ClaimHandler struct {
	claimService ClaimService
}

func NewClaimHandler(svc ClaimService) *ClaimHandler {
	return &ClaimHandler{claimService: svc}
}

// Submit handles POST /api/v1/claims/{target}/{id}. Public — anyone
// (authenticated or not) can submit a claim. The target is set at route
// registration time (one route per target) so the handler doesn't need to
// read it from the URL.
func (h *ClaimHandler) Submit(target ClaimTarget) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targetID := chi.URLParam(r, "id")

		var req SubmitClaimRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := validator.Validate.Struct(&req); err != nil {
			fields := validator.GetValidationErrors(err)
			response.ValidationError(w, http.StatusBadRequest, fields)
			return
		}

		res, err := h.claimService.Submit(r.Context(), target, targetID, &req)
		if err != nil {
			switch {
			case errors.Is(err, errs.ErrNotFound):
				response.Error(w, http.StatusNotFound, "target not found")
			case errors.Is(err, errs.ErrClaimRoleNotAllowed):
				response.Error(w, http.StatusForbidden, err.Error())
			default:
				response.Error(w, http.StatusInternalServerError, "something went wrong")
			}
			return
		}
		response.Success(w, http.StatusCreated, res)
	}
}

// List handles GET /api/v1/admin/claims?type=university|college&status=pending|approved|rejected.
// Admin only. Empty type returns both universities and colleges merged.
func (h *ClaimHandler) List(w http.ResponseWriter, r *http.Request) {
	q := pagination.Parse(r)
	statusFilter := r.URL.Query().Get("status")
	target := ClaimTarget(r.URL.Query().Get("type"))
	if target != "" && !target.IsValid() {
		response.Error(w, http.StatusBadRequest, "type must be one of: university, college")
		return
	}

	items, total, err := h.claimService.List(r.Context(), target, statusFilter, q)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrBadRequest):
			response.Error(w, http.StatusBadRequest, "status must be one of: pending, approved, rejected")
		default:
			response.Error(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	response.Success(w, http.StatusOK, pagination.Response[ClaimListItem]{
		Items: items,
		Meta:  q.BuildMeta(total),
	})
}

// GetByID handles GET /api/v1/admin/claims/{id}. Admin only.
func (h *ClaimHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	detail, err := h.claimService.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrNotFound):
			response.Error(w, http.StatusNotFound, "claim not found")
		default:
			response.Error(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}
	response.Success(w, http.StatusOK, detail)
}

// Approve handles POST /api/v1/admin/claims/{id}/approve. Admin only.
// Returns the new representative's email + a one-time plaintext password.
func (h *ClaimHandler) Approve(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req ReviewDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validator.Validate.Struct(&req); err != nil {
		fields := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, fields)
		return
	}

	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}

	res, err := h.claimService.Approve(r.Context(), id, claims.UserID, &req)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrNotFound):
			response.Error(w, http.StatusNotFound, "claim not found")
		case errors.Is(err, errs.ErrClaimAlreadyReviewed):
			response.Error(w, http.StatusConflict, err.Error())
		case errors.Is(err, errs.ErrUniversityAlreadyHasRepresentative):
			response.Error(w, http.StatusConflict, err.Error())
		case errors.Is(err, errs.ErrCollegeAlreadyHasRepresentative):
			response.Error(w, http.StatusConflict, err.Error())
		default:
			response.Error(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}
	response.Success(w, http.StatusOK, res)
}

// Reject handles POST /api/v1/admin/claims/{id}/reject. Admin only.
func (h *ClaimHandler) Reject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req ReviewDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validator.Validate.Struct(&req); err != nil {
		fields := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, fields)
		return
	}

	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}

	res, err := h.claimService.Reject(r.Context(), id, claims.UserID, &req)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrNotFound):
			response.Error(w, http.StatusNotFound, "claim not found")
		case errors.Is(err, errs.ErrClaimAlreadyReviewed):
			response.Error(w, http.StatusConflict, err.Error())
		default:
			response.Error(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}
	response.Success(w, http.StatusOK, res)
}
