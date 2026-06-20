package university

import (
	"encoding/json"
	"fmu-backend/internal/response"
	"fmu-backend/internal/validator"
	"net/http"
)

type UniversityHandler struct {
	universityService UniversityService
}

func NewUniversityHandler(universityService UniversityService) *UniversityHandler {
	return &UniversityHandler{
		universityService: universityService,
	}
}

func (h *UniversityHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUniversityRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		validationErrors := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, validationErrors)
		return
	}

	res, err := h.universityService.Create(r.Context(), &req)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusCreated, res)
}
