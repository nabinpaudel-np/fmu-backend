package auth

import (
	"encoding/json"
	"errors"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/response"
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

	res, err := h.authService.Register(r.Context(), &req)
	if err != nil {
		if errors.Is(err, errs.ErrUserAlreadyExists) {
			response.Error(w, http.StatusConflict, "user already exists")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusCreated, res)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		validationErrors := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, validationErrors)
		return
	}

	userAgent := r.Header.Get("User-Agent")

	res, err := h.authService.Login(r.Context(), &req, userAgent)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidCredentials) {
			response.Error(w, http.StatusUnauthorized, "invalid credentials")
			return
		}

		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusOK, res)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		validationErrors := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, validationErrors)
		return
	}

	userAgent := r.Header.Get("User-Agent")

	res, err := h.authService.Refresh(r.Context(), &req, userAgent)

	if err != nil {
		if errors.Is(err, errs.ErrInvalidRefreshToken) ||
			errors.Is(err, errs.ErrRefreshTokenExpired) ||
			errors.Is(err, errs.ErrRefreshTokenRevoked) {
			response.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusOK, res)
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	userAgent := r.Header.Get("User-Agent")

	if code == "" {
		url := h.authService.GetGoogleAuthURL()
		http.Redirect(w, r, url, http.StatusFound)
		return
	}

	res, err := h.authService.GoogleLogin(r.Context(), code, userAgent)
	if err != nil {
		if errors.Is(err, errs.ErrEmailAlreadyRegistered) {
			response.Error(w, http.StatusConflict, "email already registered with password login")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusOK, res)
}

func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	userAgent := r.Header.Get("User-Agent")

	if code == "" {
		response.Error(w, http.StatusBadRequest, "missing code")
		return
	}

	res, err := h.authService.GoogleLogin(r.Context(), code, userAgent)
	if err != nil {
		if errors.Is(err, errs.ErrEmailAlreadyRegistered) {
			response.Error(w, http.StatusConflict, "email already registered with password login")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusOK, res)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.authService.Logout(r.Context(), req.RefreshToken); err != nil {
		if errors.Is(err, errs.ErrInvalidRefreshToken) ||
			errors.Is(err, errs.ErrRefreshTokenExpired) ||
			errors.Is(err, errs.ErrRefreshTokenRevoked) {
			response.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusOK, nil)
}
