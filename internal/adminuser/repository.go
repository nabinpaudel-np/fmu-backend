package adminuser

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"fmu-backend/internal/db/sqlc"
)

// Repository is the slice of the user store this package needs. It is kept
// narrow so the service can be tested with a fake.
type Repository interface {
	// List returns the users matching the filters, newest first. Each row
	// carries the name of the institution a representative is bound to.
	List(ctx context.Context, role, search string, limit, offset int32) ([]sqlc.ListUsersRow, error)

	// Count returns how many users match the filters.
	Count(ctx context.Context, role, search string) (int64, error)

	// GetByID returns one user with the name of the institution a
	// representative is bound to. It returns nil when there is no such user.
	GetByID(ctx context.Context, id string) (*sqlc.GetAdminUserByIDRow, error)

	// CountAdmins returns how many admin accounts exist.
	CountAdmins(ctx context.Context) (int64, error)

	// UpdateRole changes a user's role and the institution they are bound to.
	// The returned row carries the name of that institution too. An empty
	// scope clears the binding.
	UpdateRole(ctx context.Context, id, role string, universityID, collegeID string) (*sqlc.UpdateUserRoleRow, error)

	// Delete removes the user.
	Delete(ctx context.Context, id string) error

	// InvalidateResetTokens burns every unused reset token for the user.
	InvalidateResetTokens(ctx context.Context, userID string) error
}

type repository struct {
	queries *sqlc.Queries
}

// NewRepository builds a repository over the generated queries.
func NewRepository(queries *sqlc.Queries) Repository {
	return &repository{queries: queries}
}

func (r *repository) List(ctx context.Context, role, search string, limit, offset int32) ([]sqlc.ListUsersRow, error) {
	return r.queries.ListUsers(ctx, sqlc.ListUsersParams{
		Column1: role,
		Column2: search,
		Limit:   limit,
		Offset:  offset,
	})
}

func (r *repository) Count(ctx context.Context, role, search string) (int64, error) {
	return r.queries.CountUsers(ctx, sqlc.CountUsersParams{
		Column1: role,
		Column2: search,
	})
}

func (r *repository) CountAdmins(ctx context.Context) (int64, error) {
	return r.queries.CountAdmins(ctx)
}

func (r *repository) GetByID(ctx context.Context, id string) (*sqlc.GetAdminUserByIDRow, error) {
	row, err := r.queries.GetAdminUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *repository) UpdateRole(ctx context.Context, id, role string, universityID, collegeID string) (*sqlc.UpdateUserRoleRow, error) {
	row, err := r.queries.UpdateUserRole(ctx, sqlc.UpdateUserRoleParams{
		ID:      id,
		Role:    role,
		Column3: scopeParam(universityID),
		Column4: scopeParam(collegeID),
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// scopeParam turns a scope into what the query expects. An empty scope is a
// real NULL so the binding is cleared; anything else is a uuid string.
func scopeParam(scope string) any {
	if scope == "" {
		return nil
	}
	return scope
}

func (r *repository) Delete(ctx context.Context, id string) error {
	return r.queries.DeleteUser(ctx, id)
}

func (r *repository) InvalidateResetTokens(ctx context.Context, userID string) error {
	return r.queries.InvalidateUnusedPasswordResetTokens(ctx, userID)
}
