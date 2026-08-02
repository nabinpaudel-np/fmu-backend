package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/errs"
)

type UserRepository interface {
	Create(ctx context.Context, full_name, email, password, role string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByProvider(ctx context.Context, provider, providerID string) (*User, error)
	CreateWithOAuth(ctx context.Context, fullName, email, provider, providerID, avatar string) (*User, error)
	CreateRepresentative(ctx context.Context, fullName, email, passwordHash, universityID string) (*User, error)
	CreateCollegeRepresentative(ctx context.Context, fullName, email, passwordHash, collegeID string) (*User, error)
	UpdateProfile(ctx context.Context, id string, fullName, avatar *string) (*User, error)
}

type userRepository struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
}

func NewUserRepository(queries *sqlc.Queries, pool *pgxpool.Pool) UserRepository {
	return &userRepository{
		queries: queries,
		pool:    pool,
	}
}

func (r *userRepository) Create(ctx context.Context, full_name, email, password, role string) (*User, error) {
	row, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		FullName: full_name,
		Email:    email,
		Password: &password,
		Role:     role,
	})
	if err != nil {
		return nil, err
	}
	return toDomainUser(row), nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toDomainUser(row), nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*User, error) {
	row, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toDomainUser(row), nil
}

func (r *userRepository) GetByProvider(ctx context.Context, provider, providerID string) (*User, error) {
	row, err := r.queries.GetUserByProvider(ctx, sqlc.GetUserByProviderParams{
		Provider:   &provider,
		ProviderID: &providerID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toDomainUser(row), nil
}

func (r *userRepository) CreateWithOAuth(ctx context.Context, fullName, email, provider, providerID, avatar string) (*User, error) {
	row, err := r.queries.CreateUserWithOAuth(ctx, sqlc.CreateUserWithOAuthParams{
		FullName:   fullName,
		Email:      email,
		Provider:   &provider,
		ProviderID: &providerID,
		Avatar:     &avatar,
	})
	if err != nil {
		return nil, err
	}
	return toDomainUser(row), nil
}

func (r *userRepository) CreateRepresentative(ctx context.Context, fullName, email, passwordHash, universityID string) (*User, error) {
	row, err := r.queries.CreateRepresentativeUser(ctx, sqlc.CreateRepresentativeUserParams{
		FullName:                   fullName,
		Email:                      email,
		Password:                   &passwordHash,
		RepresentativeUniversityID: uuidFromString(universityID),
	})
	if err != nil {
		return nil, err
	}
	return toDomainUser(row), nil
}

func (r *userRepository) CreateCollegeRepresentative(ctx context.Context, fullName, email, passwordHash, collegeID string) (*User, error) {
	row, err := r.queries.CreateCollegeRepresentativeUser(ctx, sqlc.CreateCollegeRepresentativeUserParams{
		FullName:                fullName,
		Email:                   email,
		Password:                &passwordHash,
		RepresentativeCollegeID: uuidFromString(collegeID),
	})
	if err != nil {
		return nil, err
	}
	return toDomainUser(row), nil
}

// UpdateProfile patches the user's full_name and/or avatar. Email is
// intentionally NOT settable here — users must contact an admin to change
// their email. The handler rejects requests that include the email field
// before reaching this method.
//
// Hand-written SQL matches the partial-update pattern used by the
// university repository: build a dynamic SET clause from non-nil pointers
// and RETURN the row in scan order. RETURNING lists columns in the order
// the rows physically come back from the database (not SELECT *), because
// migrations have appended fields after the original CREATE TABLE.
func (r *userRepository) UpdateProfile(ctx context.Context, id string, fullName, avatar *string) (*User, error) {
	if fullName == nil && avatar == nil {
		return r.GetByID(ctx, id)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	sets := []string{}
	args := []any{}
	addSet := func(col string, val any) {
		args = append(args, val)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if fullName != nil {
		addSet("full_name", *fullName)
	}
	if avatar != nil {
		addSet("avatar", *avatar)
	}
	sets = append(sets, "updated_at = now()")
	args = append(args, id)

	sql := fmt.Sprintf(`UPDATE users SET %s WHERE id = $%d
RETURNING id, full_name, avatar, email, password, oauth_provider, oauth_id, email_verified, created_at, updated_at, role, representative_university_id, representative_college_id`,
		strings.Join(sets, ", "), len(args))

	row := sqlc.User{}
	err = tx.QueryRow(ctx, sql, args...).Scan(
		&row.ID,
		&row.FullName,
		&row.Avatar,
		&row.Email,
		&row.Password,
		&row.Provider,
		&row.ProviderID,
		&row.EmailVerified,
		&row.CreatedAt,
		&row.UpdatedAt,
		&row.Role,
		&row.RepresentativeUniversityID,
		&row.RepresentativeCollegeID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrUserNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return nil, errs.ErrUserNotFound
		}
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return toDomainUser(row), nil
}

func toDomainUser(u sqlc.User) *User {
	return &User{
		ID:                         u.ID,
		FullName:                   u.FullName,
		Email:                      u.Email,
		Password:                   u.Password,
		Provider:                   u.Provider,
		ProviderID:                 u.ProviderID,
		Avatar:                     u.Avatar,
		EmailVerified:              u.EmailVerified,
		Role:                       u.Role,
		RepresentativeUniversityID: uuidToStringPtr(u.RepresentativeUniversityID),
		RepresentativeCollegeID:    uuidToStringPtr(u.RepresentativeCollegeID),
		CreatedAt:                  u.CreatedAt,
		UpdatedAt:                  u.UpdatedAt,
	}
}

func uuidToStringPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := formatUUID(u.Bytes)
	return &s
}

func uuidFromString(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{}
	}
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}
	}
	return u
}

// formatUUID renders a 16-byte UUID in canonical 8-4-4-4-12 form.
func formatUUID(b [16]byte) string {
	hex := make([]byte, 36)
	const digits = "0123456789abcdef"
	pos := 0
	for i, by := range b {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			hex[pos] = '-'
			pos++
		}
		hex[pos] = digits[by>>4]
		hex[pos+1] = digits[by&0x0f]
		pos += 2
	}
	return string(hex)
}
