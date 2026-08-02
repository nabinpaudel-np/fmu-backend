package user

import (
	"context"
	"errors"
	"fmu-backend/internal/errs"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	Create(ctx context.Context, full_name, email, password, role string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByProvider(ctx context.Context, provider, providerID string) (*User, error)
	CreateWithOAuth(ctx context.Context, fullName, email, provider, providerID, avatar string) (*User, error)
	CreateRepresentative(ctx context.Context, fullName, email, password, universityID string) (*User, error)
	CreateCollegeRepresentative(ctx context.Context, fullName, email, password, collegeID string) (*User, error)
	UpdateProfile(ctx context.Context, id string, fullName, avatar *string) (*User, error)
	UpdatePassword(ctx context.Context, id string, plaintext string) error
}

type userService struct {
	userRepo UserRepository
}

func NewUserService(userRepo UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) Create(ctx context.Context, full_name, email, password, role string) (*User, error) {
	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		return nil, errs.ErrUserAlreadyExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return s.userRepo.Create(ctx, full_name, email, string(hashed), role)
}

func (s *userService) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

func (s *userService) GetByID(ctx context.Context, id string) (*User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *userService) GetByProvider(ctx context.Context, provider, providerID string) (*User, error) {
	return s.userRepo.GetByProvider(ctx, provider, providerID)
}

func (s *userService) CreateWithOAuth(ctx context.Context, fullName, email, provider, providerID, avatar string) (*User, error) {
	return s.userRepo.CreateWithOAuth(ctx, fullName, email, provider, providerID, avatar)
}

func (s *userService) CreateRepresentative(ctx context.Context, fullName, email, password, universityID string) (*User, error) {
	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errs.ErrUserAlreadyExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return s.userRepo.CreateRepresentative(ctx, fullName, email, string(hashed), universityID)
}

func (s *userService) CreateCollegeRepresentative(ctx context.Context, fullName, email, password, collegeID string) (*User, error) {
	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errs.ErrUserAlreadyExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.CreateCollegeRepresentative(ctx, fullName, email, string(hashed), collegeID)
	if err != nil {
		// The UNIQUE constraint on representative_college_id surfaces as 23505;
		// map it to a clear error the caller can surface to admins.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, errs.ErrCollegeAlreadyHasRepresentative
		}
		return nil, err
	}
	return user, nil
}

// UpdateProfile patches the user's editable profile fields (full_name,
// avatar). Email is deliberately not in the signature — it's not
// self-serviceable. The handler rejects requests that include the email
// field before calling this.
func (s *userService) UpdateProfile(ctx context.Context, id string, fullName, avatar *string) (*User, error) {
	return s.userRepo.UpdateProfile(ctx, id, fullName, avatar)
}

// UpdatePassword bcrypts the plaintext and stores the resulting hash.
// Plaintext never touches the DB and never leaves this function. Returns
// errs.ErrUserNotFound if the user no longer exists. Used by the
// passwordreset service after the one-time token is verified.
func (s *userService) UpdatePassword(ctx context.Context, id string, plaintext string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.userRepo.UpdatePasswordHash(ctx, id, string(hashed))
}
