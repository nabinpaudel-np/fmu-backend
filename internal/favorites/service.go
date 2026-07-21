package favorites

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"fmu-backend/internal/college"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
	"fmu-backend/internal/university"
)

type UniversityLookup interface {
	GetByID(ctx context.Context, id string) (*university.UniversityDetailResponse, error)
}

type CollegeLookup interface {
	GetByID(ctx context.Context, id string) (*college.CollegeDetailResponse, error)
}

type Service interface {
	AddUniversity(ctx context.Context, userID, universityID string) error
	RemoveUniversity(ctx context.Context, userID, universityID string) error
	ListUniversities(ctx context.Context, userID string, q pagination.Query) ([]university.UniversityListItem, int64, error)

	AddCollege(ctx context.Context, userID, collegeID string) error
	RemoveCollege(ctx context.Context, userID, collegeID string) error
	ListColleges(ctx context.Context, userID string, q pagination.Query) ([]college.CollegeListItem, int64, error)
}

type service struct {
	repo        Repository
	universities UniversityLookup
	colleges     CollegeLookup
}

func NewService(repo Repository, unis UniversityLookup, cols CollegeLookup) Service {
	return &service{repo: repo, universities: unis, colleges: cols}
}

// existsUniversity returns ErrNotFound if the university doesn't exist
// (or the id is not a valid uuid). Other errors pass through.
func (s *service) existsUniversity(ctx context.Context, id string) error {
	_, err := s.universities.GetByID(ctx, id)
	if err == nil {
		return nil
	}
	if errors.Is(err, errs.ErrNotFound) || errors.Is(err, pgx.ErrNoRows) {
		return errs.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return errs.ErrNotFound
	}
	return err
}

func (s *service) existsCollege(ctx context.Context, id string) error {
	_, err := s.colleges.GetByID(ctx, id)
	if err == nil {
		return nil
	}
	if errors.Is(err, errs.ErrNotFound) || errors.Is(err, pgx.ErrNoRows) {
		return errs.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return errs.ErrNotFound
	}
	return err
}

func (s *service) AddUniversity(ctx context.Context, userID, universityID string) error {
	if err := s.existsUniversity(ctx, universityID); err != nil {
		return err
	}
	if err := s.repo.AddUniversity(ctx, userID, universityID); err != nil {
		log.Default().Printf("favorite university user=%s uni=%s: %v", userID, universityID, err)
		return err
	}
	return nil
}

func (s *service) RemoveUniversity(ctx context.Context, userID, universityID string) error {
	if err := s.repo.RemoveUniversity(ctx, userID, universityID); err != nil {
		log.Default().Printf("unfavorite university user=%s uni=%s: %v", userID, universityID, err)
		return err
	}
	return nil
}

func (s *service) ListUniversities(ctx context.Context, userID string, q pagination.Query) ([]university.UniversityListItem, int64, error) {
	items, total, err := s.repo.ListUniversities(ctx, userID, q)
	if err != nil {
		log.Default().Printf("list favorited universities user=%s: %v", userID, err)
		return nil, 0, err
	}
	return items, total, nil
}

func (s *service) AddCollege(ctx context.Context, userID, collegeID string) error {
	if err := s.existsCollege(ctx, collegeID); err != nil {
		return err
	}
	if err := s.repo.AddCollege(ctx, userID, collegeID); err != nil {
		log.Default().Printf("favorite college user=%s college=%s: %v", userID, collegeID, err)
		return err
	}
	return nil
}

func (s *service) RemoveCollege(ctx context.Context, userID, collegeID string) error {
	if err := s.repo.RemoveCollege(ctx, userID, collegeID); err != nil {
		log.Default().Printf("unfavorite college user=%s college=%s: %v", userID, collegeID, err)
		return err
	}
	return nil
}

func (s *service) ListColleges(ctx context.Context, userID string, q pagination.Query) ([]college.CollegeListItem, int64, error) {
	items, total, err := s.repo.ListColleges(ctx, userID, q)
	if err != nil {
		log.Default().Printf("list favorited colleges user=%s: %v", userID, err)
		return nil, 0, err
	}
	return items, total, nil
}