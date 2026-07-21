package college

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
)

type CollegeRepository interface {
	Create(ctx context.Context, params sqlc.CreateCollegeParams) (sqlc.College, error)
	List(ctx context.Context, q pagination.Query, f Filters) ([]sqlc.College, int64, error)
	GetByID(ctx context.Context, id string) (sqlc.College, error)
	ListByUniversity(ctx context.Context, universityID string, q pagination.Query) ([]sqlc.College, int64, error)
	Search(ctx context.Context, q string) ([]sqlc.SearchCollegesRow, error)
}

type collegeRepository struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
}

// maxCollegeSearchResults caps the /colleges/search dropdown.
const maxCollegeSearchResults = 50

// columns are spelled out (not SELECT *) because the runtime column order
// can drift from schema.sql's declaration order after ALTER TABLE ADD COLUMN.
const collegeColumnList = "id, name, slug, university_id, overview, country, state, city, full_location, logo, created_at, updated_at"

func NewCollegeRepository(queries *sqlc.Queries, pool *pgxpool.Pool) CollegeRepository {
	return &collegeRepository{
		queries: queries,
		pool:    pool,
	}
}

func (r *collegeRepository) Create(ctx context.Context, params sqlc.CreateCollegeParams) (sqlc.College, error) {
	row, err := r.queries.CreateCollege(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			return sqlc.College{}, fmt.Errorf("%w (slug=%s)", errs.ErrCollegeSlugTaken, params.Slug)
		case errors.As(err, &pgErr) && pgErr.Code == "23503":
			return sqlc.College{}, fmt.Errorf("%w (university_id=%s)", errs.ErrCollegeUniversityNotFound, params.UniversityID)
		}
		return sqlc.College{}, err
	}
	return row, nil
}

func (r *collegeRepository) List(ctx context.Context, q pagination.Query, f Filters) ([]sqlc.College, int64, error) {
	where, args := buildCollegesWhere(f)

	var total int64
	countSQL := "SELECT COUNT(*) FROM colleges" + where
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count colleges: %w", err)
	}

	listArgs := append(append([]any{}, args...), q.Limit(), q.Offset())
	listSQL := fmt.Sprintf(
		"SELECT %s FROM colleges%s ORDER BY name LIMIT $%d OFFSET $%d",
		collegeColumnList, where, len(listArgs)-1, len(listArgs),
	)

	rows, err := r.pool.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list colleges: %w", err)
	}
	defer rows.Close()

	items, err := collectColleges(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("scan colleges: %w", err)
	}
	return items, total, nil
}

func collectColleges(rows pgx.Rows) ([]sqlc.College, error) {
	items := []sqlc.College{}
	for rows.Next() {
		var c sqlc.College
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Slug, &c.UniversityID, &c.Overview,
			&c.Country, &c.State, &c.City, &c.FullLocation, &c.Logo,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

// buildCollegesWhere returns a parameterized WHERE fragment and matching args.
// Empty Filters returns empty output. country/state/city are equality filters;
// university_id is a UUID equality. No lookup-table joins.
func buildCollegesWhere(f Filters) (string, []any) {
	var clauses []string
	var args []any

	eq := func(format, val string) {
		if val == "" {
			return
		}
		args = append(args, val)
		clauses = append(clauses, fmt.Sprintf(format, len(args)))
	}

	eq("university_id = $%d", f.UniversityID)
	eq("country = $%d", f.Country)
	eq("state = $%d", f.State)
	eq("city = $%d", f.City)

	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (r *collegeRepository) GetByID(ctx context.Context, id string) (sqlc.College, error) {
	return r.queries.GetCollegeByID(ctx, id)
}

func (r *collegeRepository) ListByUniversity(ctx context.Context, universityID string, q pagination.Query) ([]sqlc.College, int64, error) {
	total, err := r.queries.CountCollegesByUniversity(ctx, universityID)
	if err != nil {
		return nil, 0, fmt.Errorf("count colleges by university: %w", err)
	}
	rows, err := r.queries.ListCollegesByUniversity(ctx, sqlc.ListCollegesByUniversityParams{
		UniversityID: universityID,
		Limit:        int32(q.Limit()),
		Offset:       int32(q.Offset()),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list colleges by university: %w", err)
	}
	return rows, total, nil
}

func (r *collegeRepository) Search(ctx context.Context, q string) ([]sqlc.SearchCollegesRow, error) {
	return r.queries.SearchColleges(ctx, sqlc.SearchCollegesParams{
		Similarity: q,
		Limit:      int32(maxCollegeSearchResults),
	})
}
