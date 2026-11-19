package adminuser

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
	"fmu-backend/internal/passwordreset"
	"fmu-backend/internal/user"
)

// The service is the whole admin flow over user accounts: an admin lists
// them, reads one, changes a role, sends a reset link, or removes one.

// UserFinder reads a single user. The passwordreset service already has one,
// so the same contract is reused here.
type UserFinder interface {
	GetByID(ctx context.Context, id string) (*user.User, error)
}

// ResetSender starts the forgot-password flow for an account. The
// passwordreset service implements it, so an admin-triggered reset and a
// self-service one take exactly the same path.
type ResetSender interface {
	ForgotPassword(ctx context.Context, req *passwordreset.ForgotPasswordRequest) (*passwordreset.ForgotPasswordResponse, error)
}

// Service is the admin-facing user management.
type Service interface {
	List(ctx context.Context, role, search string, q pagination.Query) ([]UserListItem, int64, error)
	GetByID(ctx context.Context, id string) (*UserDetailResponse, error)
	UpdateRole(ctx context.Context, id string, req *UpdateRoleRequest) (*UserDetailResponse, error)
	ResetPassword(ctx context.Context, id string) (*ResetPasswordResponse, error)
	Delete(ctx context.Context, id, actingAdminID string) error
}

type service struct {
	repo  Repository
	users UserFinder
	reset ResetSender
}

// NewService wires the service. A nil reset sender means the admin can still
// list, read, change roles and delete accounts — only the reset link is
// unavailable.
func NewService(repo Repository, users UserFinder, reset ResetSender) Service {
	return &service{repo: repo, users: users, reset: reset}
}

// List returns the users matching the filters, newest first. The filters are
// optional: an empty role matches every role and an empty search matches
// every name and email.
func (s *service) List(ctx context.Context, role, search string, q pagination.Query) ([]UserListItem, int64, error) {
	rows, err := s.repo.List(ctx, role, search, int32(q.Limit()), int32(q.Offset()))
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.Count(ctx, role, search)
	if err != nil {
		return nil, 0, err
	}

	items := make([]UserListItem, len(rows))
	for i, row := range rows {
		items[i] = toListItem(row)
	}
	return items, total, nil
}

// GetByID returns one user.
func (s *service) GetByID(ctx context.Context, id string) (*UserDetailResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, errs.ErrUserNotFound
	}
	return &UserDetailResponse{UserListItem: toListItemFromDetail(*row)}, nil
}

// UpdateRole changes a user's role.
//
// A representative is bound to exactly one university or one college, so the
// scope is required for that role and cleared for the others. Clearing it
// matters: a stale binding keeps granting edit rights over that institution,
// and its UNIQUE constraint would stop another representative from ever
// claiming it.
//
// The last admin cannot be demoted, or nobody would be left able to manage
// the catalogue.
func (s *service) UpdateRole(ctx context.Context, id string, req *UpdateRoleRequest) (*UserDetailResponse, error) {
	role := strings.TrimSpace(req.Role)
	if !isValidRole(role) {
		return nil, errs.ErrInvalidRole
	}

	current, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, errs.ErrUserNotFound
	}

	universityID, collegeID, err := s.scopeFor(role, req)
	if err != nil {
		return nil, err
	}

	// The last admin cannot be demoted to anything else.
	if current.Role == auth.RoleAdmin && role != auth.RoleAdmin {
		admins, err := s.repo.CountAdmins(ctx)
		if err != nil {
			return nil, err
		}
		if admins <= 1 {
			return nil, errs.ErrCannotRemoveLastAdmin
		}
	}

	updated, err := s.repo.UpdateRole(ctx, id, role, universityID, collegeID)
	if err != nil {
		// A representative is bound to an institution that may already have
		// one. The UNIQUE constraints on those columns surface as 23505.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, errs.ErrRepresentativeScopeConflict
		}
		return nil, err
	}

	return &UserDetailResponse{UserListItem: toListItemFromRoleUpdate(*updated)}, nil
}

// scopeFor works out which institution a representative is bound to. Only
// the representative role keeps a scope; the other two roles get none.
//
// An empty string clears the binding. It has to reach the database as a real
// NULL: an invalid UUID would be written as all zeros and the foreign key
// would reject it.
func (s *service) scopeFor(role string, req *UpdateRoleRequest) (string, string, error) {
	if role != auth.RoleRepresentative {
		return "", "", nil
	}

	university := strings.TrimSpace(req.RepresentativeUniversityID)
	college := strings.TrimSpace(req.RepresentativeCollegeID)

	// Exactly one of the two. Both or neither leaves the account unbound.
	if (university == "") == (college == "") {
		return "", "", errs.ErrRepresentativeScopeRequired
	}

	if university != "" {
		return university, "", nil
	}
	return "", college, nil
}

// ResetPassword starts the forgot-password flow for the account, so the
// holder receives a link and chooses their own password. The admin never
// sees or sets the new password.
//
// Any earlier reset link for this account is burned first, so only the
// newest one works.
func (s *service) ResetPassword(ctx context.Context, id string) (*ResetPasswordResponse, error) {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errs.ErrUserNotFound
	}

	// An account that came in through Google has no password, so a reset
	// link would lead nowhere. The dashboard is told why.
	if u.Password == nil {
		return nil, errs.ErrCannotResetOAuthOnlyPassword
	}

	if s.reset == nil {
		return nil, errs.ErrResetUnavailable
	}

	// Burn any earlier link before making a new one.
	if err := s.repo.InvalidateResetTokens(ctx, id); err != nil {
		// Not fatal — the new link still works, but an older one may too.
		log.Printf("adminuser: invalidate reset tokens for %s: %v", id, err)
	}

	res, err := s.reset.ForgotPassword(ctx, &passwordreset.ForgotPasswordRequest{Email: u.Email})
	if err != nil {
		return nil, err
	}
	if res == nil || res.AccountType != passwordreset.AccountTypePassword {
		return nil, errs.ErrCannotResetOAuthOnlyPassword
	}

	return &ResetPasswordResponse{Sent: true, Email: u.Email}, nil
}

// Delete removes a user.
//
// Two things are refused. An admin cannot delete their own account, and the
// last admin cannot be deleted at all — either one could leave the system
// with nobody able to manage it.
//
// The rows that reference the user go with them: refresh tokens, favourites
// and the claims they created all cascade in the database.
func (s *service) Delete(ctx context.Context, id, actingAdminID string) error {
	if id == actingAdminID {
		return errs.ErrCannotDeleteSelf
	}

	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if u == nil {
		return errs.ErrUserNotFound
	}

	if u.Role == auth.RoleAdmin {
		admins, err := s.repo.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if admins <= 1 {
			return errs.ErrCannotRemoveLastAdmin
		}
	}

	return s.repo.Delete(ctx, id)
}

// isValidRole reports whether a role is one the server knows.
func isValidRole(role string) bool {
	switch role {
	case auth.RoleAdmin, auth.RoleStudent, auth.RoleRepresentative:
		return true
	}
	return false
}
