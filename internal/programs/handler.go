package programs

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
	"fmu-backend/internal/response"
	"fmu-backend/internal/validator"
)

type ProgramHandler struct {
	svc ProgramService
}

func NewProgramHandler(svc ProgramService) *ProgramHandler {
	return &ProgramHandler{svc: svc}
}

func (h *ProgramHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateProgramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validator.Validate.Struct(&req); err != nil {
		fields := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, fields)
		return
	}

	res, err := h.svc.Create(r.Context(), &req)
	if err != nil {
		if errors.Is(err, errs.ErrProgramDegreeNotFound) {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusCreated, res)
}

func (h *ProgramHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "program not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, res)
}

func (h *ProgramHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "program not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProgramHandler) List(w http.ResponseWriter, r *http.Request) {
	q := pagination.Parse(r)
	f := Filter{DegreeID: r.URL.Query().Get("degree_id")}

	items, total, err := h.svc.List(r.Context(), q, f)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, pagination.Response[ProgramResponse]{
		Items: items,
		Meta:  q.BuildMeta(total),
	})
}
