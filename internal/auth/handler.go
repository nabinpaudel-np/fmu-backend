package auth

import (
	"encoding/json"
	"errors"
	"fmu-backend/internal/config"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/response"
	"fmu-backend/internal/validator"
	"net/http"
)

type AuthHandler struct {
	authService AuthService
	cfg         *config.Config
}

func NewAuthHandler(authService AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		cfg:         cfg,
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

	SetAccessCookie(w, h.cfg, res.AccessToken)
	SetRefreshCookie(w, h.cfg, res.RefreshToken)
	response.Success(w, http.StatusOK, res)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken := GetRefreshCookie(r)
	if refreshToken == "" {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}

	userAgent := r.Header.Get("User-Agent")

	res, err := h.authService.Refresh(r.Context(), refreshToken, userAgent)

	if err != nil {
		if errors.Is(err, errs.ErrInvalidRefreshToken) ||
			errors.Is(err, errs.ErrRefreshTokenExpired) ||
			errors.Is(err, errs.ErrRefreshTokenRevoked) {
			ClearAccessCookie(w, h.cfg)
			ClearRefreshCookie(w, h.cfg)
			response.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	SetAccessCookie(w, h.cfg, res.AccessToken)
	SetRefreshCookie(w, h.cfg, res.RefreshToken)
	response.Success(w, http.StatusOK, res)
}

// GoogleExchange finishes the OAuth flow. The SPA (which initiated the
// redirect to Google and now holds the `code` from Google's callback)
// POSTs { code, code_verifier } here. The backend exchanges the code with
// Google using the verifier (PKCE), looks up or creates the user, and
// sets the same HttpOnly auth cookies that /auth/login sets. Tokens
// stay out of the response body — matching the login endpoint's
// invariant — so the SPA relies entirely on cookies via
// `credentials: 'include'`.
func (h *AuthHandler) GoogleExchange(w http.ResponseWriter, r *http.Request) {
	var req googleExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validator.Validate.Struct(&req); err != nil {
		validationErrors := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, validationErrors)
		return
	}

	userAgent := r.Header.Get("User-Agent")
	res, err := h.authService.GoogleLogin(r.Context(), req.Code, req.CodeVerifier, req.RedirectURI, userAgent)
	if err != nil {
		if errors.Is(err, errs.ErrEmailAlreadyRegistered) {
			response.Error(w, http.StatusConflict, "email already registered with password login")
			return
		}
		response.Error(w, http.StatusInternalServerError, "oauth exchange failed")
		return
	}

	SetAccessCookie(w, h.cfg, res.AccessToken)
	SetRefreshCookie(w, h.cfg, res.RefreshToken)
	response.Success(w, http.StatusOK, res)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, err := ClaimsFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}

	res, err := h.authService.Me(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, errs.ErrUserNotFound) {
			response.Error(w, http.StatusUnauthorized, "user not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusOK, res)
}

// UpdateMe patches the authenticated user's own profile (full_name and
// avatar). Email is not accepted in the body — PatchProfileRequest's
// UnmarshalJSON returns an error if `email` is present, and the validator
// catches anything else. The user id always comes from the JWT, never
// the URL, so a token holder can only ever edit their own profile.
func (h *AuthHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	claims, err := ClaimsFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}

	var req PatchProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == errs.ErrEmailCannotBeChanged.Error() {
			response.Error(w, http.StatusBadRequest, errs.ErrEmailCannotBeChanged.Error())
			return
		}
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validator.Validate.Struct(&req); err != nil {
		validationErrors := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, validationErrors)
		return
	}

	res, err := h.authService.UpdateProfile(r.Context(), claims.UserID, &req)
	if err != nil {
		if errors.Is(err, errs.ErrUserNotFound) {
			response.Error(w, http.StatusNotFound, "user not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	response.Success(w, http.StatusOK, res)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken := GetRefreshCookie(r)

	if refreshToken != "" {
		_ = h.authService.Logout(r.Context(), refreshToken)
	}

	ClearAccessCookie(w, h.cfg)
	ClearRefreshCookie(w, h.cfg)
	response.Success(w, http.StatusOK, nil)
}
