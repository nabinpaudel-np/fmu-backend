package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"fmu-backend/internal/config"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/response"
	"fmu-backend/internal/validator"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
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

// GoogleLogin starts the OAuth flow. Never accepts a `code` — that would
// let an attacker fixate a victim's session onto their auth code.
func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	state := oauth2.GenerateVerifier()
	SetOAuthStateCookie(w, h.cfg, state)
	http.Redirect(w, r, h.authService.GetGoogleAuthURL(state), http.StatusFound)
}

// GoogleCallback finishes the OAuth flow. Validates state vs the cookie
// (CSRF), exchanges the code for tokens, and clears the state cookie on
// every code path so it can't be replayed.
func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if oauthErr := q.Get("error"); oauthErr != "" {
		ClearOAuthStateCookie(w, h.cfg)
		redirectWithError(w, r, h.authService.FrontendURL(), oauthErr, q.Get("error_description"))
		return
	}

	code := q.Get("code")
	state := q.Get("state")

	if code == "" {
		ClearOAuthStateCookie(w, h.cfg)
		response.Error(w, http.StatusBadRequest, "missing code")
		return
	}

	expected := GetOAuthStateCookie(r)
	if expected == "" || expected != state {
		ClearOAuthStateCookie(w, h.cfg)
		response.Error(w, http.StatusUnauthorized, "invalid state")
		return
	}

	ClearOAuthStateCookie(w, h.cfg)

	userAgent := r.Header.Get("User-Agent")
	res, err := h.authService.GoogleLogin(r.Context(), code, state, userAgent)
	if err != nil {
		if errors.Is(err, errs.ErrEmailAlreadyRegistered) {
			redirectWithError(w, r, h.authService.FrontendURL(), "email_taken", "this email is already registered with password login")
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	SetAccessCookie(w, h.cfg, res.AccessToken)
	SetRefreshCookie(w, h.cfg, res.RefreshToken)
	http.Redirect(w, r, h.authService.FrontendURL(), http.StatusFound)
}

func redirectWithError(w http.ResponseWriter, r *http.Request, frontendURL, errCode, errDescription string) {
	redirect := frontendURL
	if u, err := url.Parse(frontendURL); err == nil {
		q := u.Query()
		q.Set("error", errCode)
		if errDescription != "" {
			q.Set("error_description", errDescription)
		}
		u.RawQuery = q.Encode()
		redirect = u.String()
	} else {
		redirect = fmt.Sprintf("%s?error=%s&error_description=%s", frontendURL, url.QueryEscape(errCode), url.QueryEscape(errDescription))
	}
	http.Redirect(w, r, redirect, http.StatusFound)
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

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken := GetRefreshCookie(r)

	if refreshToken != "" {
		_ = h.authService.Logout(r.Context(), refreshToken)
	}

	ClearAccessCookie(w, h.cfg)
	ClearRefreshCookie(w, h.cfg)
	response.Success(w, http.StatusOK, nil)
}
