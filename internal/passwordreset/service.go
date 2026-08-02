package passwordreset

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"strings"
	"time"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/config"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/mail"
	"fmu-backend/internal/token"
	"fmu-backend/internal/user"
)

// UserUpdaters is the slice of the user service the password-reset flow
// needs — keep this small so we can wire in tests without standing up the
// whole user package.
type UserUpdaters interface {
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	GetByID(ctx context.Context, id string) (*user.User, error)
	UpdatePassword(ctx context.Context, id string, plaintext string) error
}

// TokenRevoker revokes all existing refresh tokens for a user — used after
// a successful password reset to invalidate sessions on every device.
type TokenRevoker interface {
	RevokeAllForUser(ctx context.Context, userID string) error
	CreateAccessToken(userID, email, role, repUniversityID, repCollegeID string) (string, error)
	CreateRefreshToken(ctx context.Context, userID string, userAgent string) (string, error)
}

// ResetTokenIssuer is the slim repo contract used here.
type ResetTokenIssuer interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*PasswordResetToken, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	MarkUsed(ctx context.Context, id string) error
}

type Service interface {
	ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) (*ForgotPasswordResponse, error)
	ResetPassword(ctx context.Context, req *ResetPasswordRequest, userAgent string) (*auth.LoginResponse, error)
}

type service struct {
	repo       ResetTokenIssuer
	users      UserUpdaters
	tokens     TokenRevoker
	mailer     *mail.Client
	cfg        *config.Config
	expiry     time.Duration
	resetURLFn func(token string) string // injected so the email link is easy to override in tests
}

// NewService wires the password-reset service. resetURLFn builds the link
// that lands the user on the frontend reset page (the token is appended as
// a query param); callers typically derive it from cfg.FrontendURL.
//
// Mailer is optional — when nil (SMTP not configured), forgot-password
// still succeeds but no email is sent. Reset still succeeds in dev too;
// without a mailer the token never reaches the user, so callers usually
// rely on direct DB seeding in that mode.
func NewService(repo ResetTokenIssuer, users UserUpdaters, tokens TokenRevoker, mailer *mail.Client, cfg *config.Config, resetURLFn func(string) string) Service {
	if resetURLFn == nil {
		resetURLFn = func(token string) string {
			return strings.TrimRight(cfg.FrontendURL, "/") + "/reset-password?token=" + token
		}
	}
	return &service{
		repo:       repo,
		users:      users,
		tokens:     tokens,
		mailer:     mailer,
		cfg:        cfg,
		expiry:     cfg.PasswordResetExpiry,
		resetURLFn: resetURLFn,
	}
}

// ForgotPassword handles POST /auth/forgot-password. Returns an
// account-type flag the frontend uses to pick the next screen:
//   - "password" → email was sent (user has a password on file)
//   - "oauth"    → no email (OAuth-only user — frontend should offer
//     "sign in with Google" instead)
//   - "none"     → no such account (frontend shows a generic "if an
//     account exists you'll receive an email" message)
//
// The response intentionally surfaces account existence to the caller so
// the SPA can drive a clean UX. This is the deliberate trade-off — see
// the comment on AccountType in dto.go and the API.md auth section.
func (s *service) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) (*ForgotPasswordResponse, error) {
	u, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		log.Default().Printf("passwordreset: lookup %s: %v", req.Email, err)
		// Don't surface DB errors to the caller — return "none" so the
		// response is indistinguishable from an unknown email.
		return &ForgotPasswordResponse{AccountType: AccountTypeNone}, nil
	}
	if u == nil {
		return &ForgotPasswordResponse{AccountType: AccountTypeNone}, nil
	}
	if u.Password == nil {
		// OAuth-only account — let the frontend offer "sign in with Google"
		// instead of a reset email we can't act on.
		return &ForgotPasswordResponse{AccountType: AccountTypeOAuth}, nil
	}

	plaintext, err := generateToken()
	if err != nil {
		log.Default().Printf("passwordreset: generate token: %v", err)
		return nil, err
	}
	hash := token.HashRefreshToken(plaintext) // SHA-256 hex — reused primitive
	expiresAt := time.Now().Add(s.expiry)
	if _, err := s.repo.Create(ctx, u.ID, hash, expiresAt); err != nil {
		log.Default().Printf("passwordreset: persist token: %v", err)
		return nil, err
	}

	if s.mailer != nil {
		link := s.resetURLFn(plaintext)
		if err := s.mailer.SendPasswordReset(ctx, mail.PasswordResetData{
			FullName:      u.FullName,
			Email:         u.Email,
			ResetURL:      link,
			ExpiryMinutes: int(s.expiry.Minutes()),
		}); err != nil {
			// Best-effort — the token still works if the admin can hand it
			// over manually. Don't fail the request because SMTP blipped.
			log.Default().Printf("passwordreset: send email: %v", err)
		}
	}

	return &ForgotPasswordResponse{AccountType: AccountTypePassword}, nil
}

// ResetPassword handles POST /auth/reset-password. On success issues a
// fresh login session (access + refresh tokens + user payload) so the SPA
// can drop the user straight into the app after reset.
func (s *service) ResetPassword(ctx context.Context, req *ResetPasswordRequest, userAgent string) (*auth.LoginResponse, error) {
	hash := token.HashRefreshToken(req.Token)

	reset, err := s.repo.GetByTokenHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if reset == nil {
		return nil, errs.ErrInvalidPasswordResetToken
	}
	if reset.UsedAt != nil {
		return nil, errs.ErrPasswordResetTokenUsed
	}
	if time.Now().After(reset.ExpiresAt) {
		return nil, errs.ErrPasswordResetTokenExpired
	}

	u, err := s.users.GetByID(ctx, reset.UserID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		// Token valid but user gone — treat as invalid token rather than 404
		// (don't leak whether the user exists).
		return nil, errs.ErrInvalidPasswordResetToken
	}

	// Mark the token used FIRST so a stale replay during a slow password
	// update can't double-consume. Sequence is: mark → mint new session →
	// rotate password → revoke legacy sessions. Each step can fail; the
	// caller sees a 500 and the partially-completed steps are bounded:
	//   - If mark fails → no side effects.
	//   - If mint fails → token marked used, password unchanged. User must
	//     request a fresh reset (annoying but recoverable).
	//   - If password update fails → token marked used, no new session, no
	//     legacy revoke. User must request a fresh reset.
	//   - If revoke fails → logged and ignored; new tokens are already
	//     valid, old ones will keep working until their TTL.
	if err = s.repo.MarkUsed(ctx, reset.ID); err != nil {
		log.Default().Printf("passwordreset: mark used %s: %v", reset.ID, err)
		return nil, err
	}

	access, refresh, err := mintSession(ctx, s.tokens, u, userAgent)
	if err != nil {
		return nil, err
	}

	if err = s.users.UpdatePassword(ctx, u.ID, req.Password); err != nil {
		if errors.Is(err, errs.ErrUserNotFound) {
			log.Default().Printf("passwordreset: user %s vanished mid-reset", u.ID)
			return nil, errs.ErrInvalidPasswordResetToken
		}
		return nil, err
	}

	if err = s.tokens.RevokeAllForUser(ctx, u.ID); err != nil {
		// Non-fatal — the new tokens we just minted are valid, but legacy
		// sessions on other devices are still alive. Log and continue.
		log.Default().Printf("passwordreset: revoke legacy refresh tokens for %s: %v", u.ID, err)
	}

	return toLoginResponse(u, access, refresh), nil
}

// mintSession creates the access + refresh token pair for the freshly-reset
// account. Pulled out so the SQL transaction in ResetPassword can roll back
// cleanly on any partial failure.
func mintSession(ctx context.Context, tk TokenRevoker, u *user.User, userAgent string) (string, string, error) {
	repUni := ""
	if u.RepresentativeUniversityID != nil {
		repUni = *u.RepresentativeUniversityID
	}
	repCol := ""
	if u.RepresentativeCollegeID != nil {
		repCol = *u.RepresentativeCollegeID
	}

	access, err := tk.CreateAccessToken(u.ID, u.Email, u.Role, repUni, repCol)
	if err != nil {
		return "", "", err
	}
	refresh, err := tk.CreateRefreshToken(ctx, u.ID, userAgent)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func toLoginResponse(u *user.User, access, refresh string) *auth.LoginResponse {
	avatar := ""
	if u.Avatar != nil {
		avatar = *u.Avatar
	}
	repUni := ""
	if u.RepresentativeUniversityID != nil {
		repUni = *u.RepresentativeUniversityID
	}
	repCol := ""
	if u.RepresentativeCollegeID != nil {
		repCol = *u.RepresentativeCollegeID
	}
	return &auth.LoginResponse{
		AccessToken:                access,
		RefreshToken:               refresh,
		UserID:                     u.ID,
		FullName:                   u.FullName,
		Email:                      u.Email,
		Avatar:                     avatar,
		Role:                       u.Role,
		RepresentativeUniversityID: repUni,
		RepresentativeCollegeID:    repCol,
	}
}

// generateToken returns a URL-safe base64 of 32 random bytes. Stable on
// both sides of the hash (the link the user clicks holds this exact
// string; the DB sees only sha256(plaintext)).
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
