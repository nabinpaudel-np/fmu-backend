package studentprofile

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
	"fmu-backend/internal/response"
	"fmu-backend/internal/university"
	"fmu-backend/internal/validator"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// userID extracts the authenticated user's id from request context. The
// authMW guarantees claims are present; if not, something upstream is
// mis-wired.
func userID(r *http.Request) (string, error) {
	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validator.Validate.Struct(&req); err != nil {
		response.ValidationError(w, http.StatusBadRequest, validator.GetValidationErrors(err))
		return
	}

	res, err := h.svc.Upsert(r.Context(), uid, &req)
	if err != nil {
		if errors.Is(err, errs.ErrBadRequest) {
			response.Error(w, http.StatusBadRequest, "request body must include at least one of: budget, intended_country, current_education, program_ids")
			return
		}
		var invalidRefs *InvalidProgramRefsError
		if errors.As(err, &invalidRefs) {
			response.ValidationError(w, http.StatusBadRequest, []response.ErrorDetail{{
				Field:   "program_ids",
				Message: fmt.Sprintf("the following program_ids do not exist: [%s]", strings.Join(invalidRefs.Missing, ", ")),
			}})
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusOK, res)
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}

	res, err := h.svc.Get(r.Context(), uid)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "no profile set")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusOK, res)
}

func (h *Handler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}

	if err := h.svc.Delete(r.Context(), uid); err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusOK, nil)
}

func (h *Handler) ListRecommendations(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}

	q := pagination.Parse(r)
	items, total, err := h.svc.ListRecommendations(r.Context(), uid, q)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			response.Error(w, http.StatusBadRequest, "set your preferences before requesting recommendations")
			return
		}
		if errors.Is(err, errs.ErrBadRequest) {
			response.Error(w, http.StatusBadRequest, "set your preferences before requesting recommendations")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusOK, pagination.Response[university.UniversityListItem]{
		Items: items,
		Meta:  q.BuildMeta(total),
	})
}