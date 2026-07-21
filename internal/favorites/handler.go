package favorites

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
	"fmu-backend/internal/response"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// userID extracts the authenticated user's id from request context. AuthMW
// guarantees claims are present; if not, something upstream is broken.
func userID(r *http.Request) (string, error) {
	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errs.ErrNotFound):
		response.Error(w, http.StatusNotFound, "not found")
	default:
		response.Error(w, http.StatusInternalServerError, "something went wrong")
	}
}

func (h *Handler) FavoriteUniversity(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}
	if err := h.svc.AddUniversity(r.Context(), uid, chi.URLParam(r, "id")); err != nil {
		writeErr(w, err)
		return
	}
	response.Success(w, http.StatusOK, nil)
}

func (h *Handler) UnfavoriteUniversity(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}
	if err := h.svc.RemoveUniversity(r.Context(), uid, chi.URLParam(r, "id")); err != nil {
		writeErr(w, err)
		return
	}
	response.Success(w, http.StatusOK, nil)
}

func (h *Handler) ListUniversities(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}
	q := pagination.Parse(r)
	items, total, err := h.svc.ListUniversities(r.Context(), uid, q)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, pagination.Response[UniversityListItem]{
		Items: items,
		Meta:  q.BuildMeta(total),
	})
}

func (h *Handler) FavoriteCollege(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}
	if err := h.svc.AddCollege(r.Context(), uid, chi.URLParam(r, "id")); err != nil {
		writeErr(w, err)
		return
	}
	response.Success(w, http.StatusOK, nil)
}

func (h *Handler) UnfavoriteCollege(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}
	if err := h.svc.RemoveCollege(r.Context(), uid, chi.URLParam(r, "id")); err != nil {
		writeErr(w, err)
		return
	}
	response.Success(w, http.StatusOK, nil)
}

func (h *Handler) ListColleges(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}
	q := pagination.Parse(r)
	items, total, err := h.svc.ListColleges(r.Context(), uid, q)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, pagination.Response[CollegeListItem]{
		Items: items,
		Meta:  q.BuildMeta(total),
	})
}