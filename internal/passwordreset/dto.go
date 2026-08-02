package passwordreset

// AccountType tells the frontend how to render the forgot-password flow.
// Returned in the response body so the SPA can branch:
//   "password" → user has a password, email was sent
//   "oauth"    → user signed up via Google only, no reset email — show "sign in with Google"
//   "none"     → no such account; generic "if an account exists, you'll receive an email" copy
//
// Deliberately leaks account existence to the caller — the frontend uses
// it to pick a good UX rather than silently dropping OAuth users. The
// trade-off is documented in API.md.
type AccountType string

const (
	AccountTypeNone     AccountType = "none"
	AccountTypePassword AccountType = "password"
	AccountTypeOAuth    AccountType = "oauth"
)

// ForgotPasswordRequest is the body for POST /api/v1/auth/forgot-password.
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email" example:"ada@example.com"`
}

// ForgotPasswordResponse tells the frontend which flow to show next. The
// email was actually sent only when AccountType == "password".
type ForgotPasswordResponse struct {
	AccountType AccountType `json:"account_type" example:"password"`
}

// ResetPasswordRequest is the body for POST /api/v1/auth/reset-password.
// On success, the response also issues a fresh login session for the user
// (access + refresh cookies + LoginResponse data) — no extra /login call.
type ResetPasswordRequest struct {
	Token    string `json:"token"    validate:"required,min=10"  example:"EBrg9...long-random-string..."`
	Password string `json:"password" validate:"required,min=8"   example:"correct-horse-battery-staple"`
}
