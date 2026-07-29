package programs

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
)

type Filter struct {
	// DegreeID filters to programs under one degree. Empty = no filter.
	DegreeID string
}

func (f Filter) Empty() bool { return f.DegreeID == "" }

type ProgramRepository interface {
	Create(ctx context.Context, params sqlc.CreateProgramParams) (sqlc.Program, error)
	GetByID(ctx context.Context, id string) (sqlc.Program, error)
	Update(ctx context.Context, params sqlc.UpdateProgramParams) (sqlc.Program, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, q pagination.Query, f Filter) ([]sqlc.Program, int64, error)
	// ListAll returns every program sorted by title — used by the /lookups
	// bundle. Unbounded by design; the admin-curated programs table is small.
	ListAll(ctx context.Context) ([]sqlc.Program, error)
}

type programRepository struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
}

func NewProgramRepository(queries *sqlc.Queries, pool *pgxpool.Pool) ProgramRepository {
	return &programRepository{queries: queries, pool: pool}
}

func (r *programRepository) Create(ctx context.Context, params sqlc.CreateProgramParams) (sqlc.Program, error) {
	row, err := r.queries.CreateProgram(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return sqlc.Program{}, fmt.Errorf("%w (degree_id=%s)", errs.ErrProgramDegreeNotFound, params.DegreeID)
		}
		return sqlc.Program{}, err
	}
	return row, nil
}

func (r *programRepository) GetByID(ctx context.Context, id string) (sqlc.Program, error) {
	row, err := r.queries.GetProgramByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Program{}, errs.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return sqlc.Program{}, errs.ErrNotFound
		}
		return sqlc.Program{}, err
	}
	return row, nil
}

func (r *programRepository) Update(ctx context.Context, params sqlc.UpdateProgramParams) (sqlc.Program, error) {
	row, err := r.queries.UpdateProgram(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Program{}, errs.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return sqlc.Program{}, errs.ErrNotFound
		}
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return sqlc.Program{}, fmt.Errorf("%w (degree_id=%s)", errs.ErrProgramDegreeNotFound, params.DegreeID)
		}
		return sqlc.Program{}, err
	}
	return row, nil
}

func (r *programRepository) Delete(ctx context.Context, id string) error {
	rows, err := r.queries.DeleteProgram(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return errs.ErrNotFound
	}
	return nil
}

func (r *programRepository) List(ctx context.Context, q pagination.Query, f Filter) ([]sqlc.Program, int64, error) {
	if f.DegreeID != "" {
		total, err := r.queries.CountProgramsByDegree(ctx, f.DegreeID)
		if err != nil {
			return nil, 0, fmt.Errorf("count programs by degree: %w", err)
		}
		rows, err := r.queries.ListProgramsByDegree(ctx, sqlc.ListProgramsByDegreeParams{
			DegreeID: f.DegreeID,
			Limit:    int32(q.Limit()),
			Offset:   int32(q.Offset()),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("list programs by degree: %w", err)
		}
		return rows, total, nil
	}

	total, err := r.queries.CountPrograms(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count programs: %w", err)
	}
	rows, err := r.queries.ListPrograms(ctx, sqlc.ListProgramsParams{
		Limit:  int32(q.Limit()),
		Offset: int32(q.Offset()),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list programs: %w", err)
	}
	return rows, total, nil
}

func (r *programRepository) ListAll(ctx context.Context) ([]sqlc.Program, error) {
	rows, err := r.queries.ListAllPrograms(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all programs: %w", err)
	}
	return rows, nil
}
