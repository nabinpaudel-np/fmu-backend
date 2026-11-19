package adminuser

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
	"fmu-backend/internal/response"
	"fmu-backend/internal/validator"
)

type Handler struct {
	service Service
}

// NewHandler builds the admin user handler.
func NewHandler(svc Service) *Handler {
	return &Handler{service: svc}
}

// List handles GET /api/v1/admin/users?role=&search=&page=&page_size=.
// Admin only. An empty role lists every role and an empty search lists
// every account; the search matches a name or an email, case-insensitively.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := pagination.Parse(r)

	role := strings.TrimSpace(r.URL.Query().Get("role"))
	if role != "" && !isValidRole(role) {
		response.Error(w, http.StatusBadRequest, errs.ErrInvalidRole.Error())
		return
	}
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	items, total, err := h.service.List(r.Context(), role, search, q)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusOK, pagination.Response[UserListItem]{
		Items: items,
		Meta:  q.BuildMeta(total),
	})
}

// GetByID handles GET /api/v1/admin/users/{id}. Admin only.
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	detail, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrUserNotFound):
			response.Error(w, http.StatusNotFound, "user not found")
		default:
			response.Error(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}
	response.Success(w, http.StatusOK, detail)
}

// UpdateRole handles PATCH /api/v1/admin/users/{id}/role. Admin only.
//
// The representative role needs the institution the account is bound to; the
// other two roles keep none. Changing away from "representative" clears the
// binding so the institution is free for someone else to claim.
func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validator.Validate.Struct(&req); err != nil {
		fields := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, fields)
		return
	}

	res, err := h.service.UpdateRole(r.Context(), id, &req)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrUserNotFound):
			response.Error(w, http.StatusNotFound, "user not found")
		case errors.Is(err, errs.ErrInvalidRole):
			response.Error(w, http.StatusBadRequest, errs.ErrInvalidRole.Error())
		case errors.Is(err, errs.ErrRepresentativeScopeRequired):
			response.Error(w, http.StatusBadRequest, errs.ErrRepresentativeScopeRequired.Error())
		case errors.Is(err, errs.ErrRepresentativeScopeConflict):
			response.Error(w, http.StatusConflict, errs.ErrRepresentativeScopeConflict.Error())
		case errors.Is(err, errs.ErrCannotRemoveLastAdmin):
			response.Error(w, http.StatusConflict, errs.ErrCannotRemoveLastAdmin.Error())
		default:
			log.Printf("adminuser: update role for %s: %v", id, err)
			response.Error(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}
	response.Success(w, http.StatusOK, res)
}

// ResetPassword handles POST /api/v1/admin/users/{id}/reset-password.
// Admin only.
//
// The account holder is emailed a link and chooses their own password, so
// the admin never sees or sets it.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	res, err := h.service.ResetPassword(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrUserNotFound):
			response.Error(w, http.StatusNotFound, "user not found")
		case errors.Is(err, errs.ErrCannotResetOAuthOnlyPassword):
			response.Error(w, http.StatusConflict, errs.ErrCannotResetOAuthOnlyPassword.Error())
		case errors.Is(err, errs.ErrResetUnavailable):
			response.Error(w, http.StatusServiceUnavailable, errs.ErrResetUnavailable.Error())
		default:
			response.Error(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}
	response.Success(w, http.StatusOK, res)
}

// Delete handles DELETE /api/v1/admin/users/{id}. Admin only.
//
// An admin cannot delete their own account, and the last admin cannot be
// deleted at all.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}

	if err := h.service.Delete(r.Context(), id, claims.UserID); err != nil {
		switch {
		case errors.Is(err, errs.ErrUserNotFound):
			response.Error(w, http.StatusNotFound, "user not found")
		case errors.Is(err, errs.ErrCannotDeleteSelf):
			response.Error(w, http.StatusConflict, errs.ErrCannotDeleteSelf.Error())
		case errors.Is(err, errs.ErrCannotRemoveLastAdmin):
			response.Error(w, http.StatusConflict, errs.ErrCannotRemoveLastAdmin.Error())
		default:
			response.Error(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}
	response.Success(w, http.StatusOK, nil)
}
