package programs

import (
	"context"
	"log"

	"fmu-backend/internal/pagination"
)

type ProgramService interface {
	Create(ctx context.Context, req *CreateProgramRequest) (*ProgramResponse, error)
	GetByID(ctx context.Context, id string) (*ProgramResponse, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, q pagination.Query, f Filter) ([]ProgramResponse, int64, error)
	// ListAll is the unbounded list used by the /lookups bundle. Sorted by
	// title so the dropdown is stable.
	ListAll(ctx context.Context) ([]ProgramLookupItem, error)
}

type programService struct {
	repo ProgramRepository
}

func NewProgramService(repo ProgramRepository) ProgramService {
	return &programService{repo: repo}
}

func (s *programService) Create(ctx context.Context, req *CreateProgramRequest) (*ProgramResponse, error) {
	row, err := s.repo.Create(ctx, toCreateParams(req))
	if err != nil {
		log.Default().Printf("failed to create program: %v", err)
		return nil, err
	}
	return toResponse(row), nil
}

func (s *programService) GetByID(ctx context.Context, id string) (*ProgramResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Default().Printf("failed to get program %s: %v", id, err)
		return nil, err
	}
	return toResponse(row), nil
}

func (s *programService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		log.Default().Printf("failed to delete program %s: %v", id, err)
		return err
	}
	return nil
}

func (s *programService) List(ctx context.Context, q pagination.Query, f Filter) ([]ProgramResponse, int64, error) {
	rows, total, err := s.repo.List(ctx, q, f)
	if err != nil {
		log.Default().Printf("failed to list programs: %v", err)
		return nil, 0, err
	}
	items := make([]ProgramResponse, len(rows))
	for i, row := range rows {
		items[i] = *toResponse(row)
	}
	return items, total, nil
}

func (s *programService) ListAll(ctx context.Context) ([]ProgramLookupItem, error) {
	rows, err := s.repo.ListAll(ctx)
	if err != nil {
		log.Default().Printf("failed to list programs for lookups: %v", err)
		return nil, err
	}
	items := make([]ProgramLookupItem, len(rows))
	for i, row := range rows {
		items[i] = toLookupItem(row)
	}
	return items, nil
}
