package passwordreset

import (
	"encoding/json"
	"errors"
	"net/http"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/config"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/response"
	"fmu-backend/internal/validator"
)

type Handler struct {
	svc Service
	cfg *config.Config
}

func NewHandler(svc Service, cfg *config.Config) *Handler {
	return &Handler{svc: svc, cfg: cfg}
}

// ForgotPassword handles POST /api/v1/auth/forgot-password. Always
// returns 200 with a body that names the account type, so the frontend
// can branch (real password user → "check your inbox"; OAuth-only user →
// "sign in with Google"; unknown email → generic "if an account exists"
// copy). A bad JSON body is still a 400 because that's a client error,
// not a probe response.
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validator.Validate.Struct(&req); err != nil {
		fields := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, fields)
		return
	}
	res, err := h.svc.ForgotPassword(r.Context(), &req)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	response.Success(w, http.StatusOK, res)
}

// ResetPassword handles POST /api/v1/auth/reset-password. On success
// issues the standard auth cookies and returns the same LoginResponse
// shape as /auth/login so the SPA can replace its user state and stay
// signed in without a second round trip.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validator.Validate.Struct(&req); err != nil {
		fields := validator.GetValidationErrors(err)
		response.ValidationError(w, http.StatusBadRequest, fields)
		return
	}

	userAgent := r.Header.Get("User-Agent")
	res, err := h.svc.ResetPassword(r.Context(), &req, userAgent)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrInvalidPasswordResetToken),
			errors.Is(err, errs.ErrPasswordResetTokenUsed),
			errors.Is(err, errs.ErrPasswordResetTokenExpired):
			response.Error(w, http.StatusBadRequest, err.Error())
		default:
			response.Error(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	auth.SetAccessCookie(w, h.cfg, res.AccessToken)
	auth.SetRefreshCookie(w, h.cfg, res.RefreshToken)
	response.Success(w, http.StatusOK, res)
}
