package college

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
)

type CollegeService interface {
	Create(ctx context.Context, req *CreateCollegeRequest) (*CreateCollegeResponse, error)
	Update(ctx context.Context, id string, req *UpdateCollegeRequest) (*CreateCollegeResponse, error)
	List(ctx context.Context, q pagination.Query, f Filters) ([]CollegeListItem, int64, error)
	GetByID(ctx context.Context, id string) (*CollegeDetailResponse, error)
	ListByUniversity(ctx context.Context, universityID string, q pagination.Query) ([]CollegeListItem, int64, error)
	Search(ctx context.Context, q string) ([]CollegeSearchResult, error)
}

type collegeService struct {
	repo CollegeRepository
}

func NewCollegeService(repo CollegeRepository) CollegeService {
	return &collegeService{repo: repo}
}

func (s *collegeService) Create(ctx context.Context, req *CreateCollegeRequest) (*CreateCollegeResponse, error) {
	// Representatives can create colleges only under their own university.
	// Admins are unrestricted. AuthMiddleware injects claims into ctx so
	// the body-level scope check lives next to the data write.
	if claims, err := auth.ClaimsFromContext(ctx); err == nil && claims.Role == auth.RoleRepresentative {
		if claims.RepresentativeUniversityID == "" || claims.RepresentativeUniversityID != req.UniversityID {
			return nil, errs.ErrRepOutOfScope
		}
	}

	row, err := s.repo.Create(ctx, toCreateCollegeParams(req))
	if err != nil {
		log.Default().Printf("failed to create college: %v", err)
		return nil, err
	}
	return toCreateCollegeResponse(row), nil
}

func (s *collegeService) List(ctx context.Context, q pagination.Query, f Filters) ([]CollegeListItem, int64, error) {
	rows, total, err := s.repo.List(ctx, q, f)
	if err != nil {
		log.Default().Printf("failed to list colleges: %v", err)
		return nil, 0, err
	}
	items := make([]CollegeListItem, len(rows))
	for i, row := range rows {
		items[i] = toCollegeListItem(row)
	}
	return items, total, nil
}

func (s *collegeService) GetByID(ctx context.Context, id string) (*CollegeDetailResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return nil, errs.ErrNotFound
		}
		log.Default().Printf("failed to get college %s: %v", id, err)
		return nil, err
	}
	return toCollegeDetailResponse(row), nil
}

func (s *collegeService) Update(ctx context.Context, id string, req *UpdateCollegeRequest) (*CreateCollegeResponse, error) {
	// AuthMiddleware injects claims into ctx; admin-or-rep middleware already
	// gated the route. For reps we additionally require their bound college
	// id to match the URL — RequireCollegeEditor handles admin + rep-scope
	// checks at the HTTP layer, so by the time we reach here, the only
	// remaining check is "did the caller pass an id they own?" — already
	// validated upstream. No additional in-body guard needed.
	row, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		if errors.Is(err, errs.ErrCollegeSlugTaken) {
			return nil, err
		}
		log.Default().Printf("failed to update college %s: %v", id, err)
		return nil, err
	}
	return toCreateCollegeResponse(row), nil
}

func (s *collegeService) ListByUniversity(ctx context.Context, universityID string, q pagination.Query) ([]CollegeListItem, int64, error) {
	rows, total, err := s.repo.ListByUniversity(ctx, universityID, q)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return nil, 0, errs.ErrNotFound
		}
		log.Default().Printf("failed to list colleges for university %s: %v", universityID, err)
		return nil, 0, err
	}
	items := make([]CollegeListItem, len(rows))
	for i, row := range rows {
		items[i] = toCollegeListItem(row)
	}
	return items, total, nil
}

func (s *collegeService) Search(ctx context.Context, q string) ([]CollegeSearchResult, error) {
	rows, err := s.repo.Search(ctx, q)
	if err != nil {
		log.Default().Printf("search colleges q=%q: %v", q, err)
		return nil, err
	}
	items := make([]CollegeSearchResult, len(rows))
	for i, row := range rows {
		items[i] = toCollegeSearchResult(row)
	}
	return items, nil
}
