package auth

import (
	"encoding/json"
	"errors"
	"fmu-backend/internal/response"
	"fmu-backend/internal/user"
	"fmu-backend/internal/validator"
	"net/http"
)

type AuthHandler struct {
	authService AuthService
}

func NewAuthHandler(authService AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		validationErrors := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, validationErrors)
		return
	}

	u, err := h.authService.Register(r.Context(), &req)
	if err != nil {
		if errors.Is(err, user.ErrUserAlreadyExists) {
			response.Error(w, http.StatusConflict, "user already exists")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusCreated, u)
}
