package studentprofile

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
	"fmu-backend/internal/university"
)

// Repository is the persistence boundary for student profiles and the
// derived /recommendations list. The recommendation query is hand-rolled
// SQL (see listRecommendedSQL) — too custom to express as a single sqlc
// query and must scan into the shared university.UniversityListItem
// type, so the column list is fixed in code rather than via :many.
type Repository interface {
	Upsert(ctx context.Context, userID string, p sqlc.StudentProfile) (sqlc.StudentProfile, error)
	Get(ctx context.Context, userID string) (sqlc.StudentProfile, error)
	Delete(ctx context.Context, userID string) error

	ReplaceProgramIDs(ctx context.Context, userID string, ids []string) error
	ListProgramIDs(ctx context.Context, userID string) ([]string, error)
	ExistingProgramIDs(ctx context.Context, ids []string) (map[string]struct{}, error)

	ListRecommendedUniversities(ctx context.Context, profile sqlc.StudentProfile, programIDs []string, q pagination.Query) ([]university.UniversityListItem, int64, error)
}

type repository struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
}

func NewRepository(queries *sqlc.Queries, pool *pgxpool.Pool) Repository {
	return &repository{queries: queries, pool: pool}
}

func (r *repository) Upsert(ctx context.Context, userID string, p sqlc.StudentProfile) (sqlc.StudentProfile, error) {
	row, err := r.queries.UpsertStudentProfile(ctx, sqlc.UpsertStudentProfileParams{
		UserID:           userID,
		Budget:           p.Budget,
		IntendedCountry:  p.IntendedCountry,
		CurrentEducation: p.CurrentEducation,
	})
	if err != nil {
		return sqlc.StudentProfile{}, fmt.Errorf("upsert student profile: %w", err)
	}
	return row, nil
}

func (r *repository) Get(ctx context.Context, userID string) (sqlc.StudentProfile, error) {
	row, err := r.queries.GetStudentProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.StudentProfile{}, errs.ErrNotFound
		}
		return sqlc.StudentProfile{}, fmt.Errorf("get student profile: %w", err)
	}
	return row, nil
}

func (r *repository) Delete(ctx context.Context, userID string) error {
	if err := r.queries.DeleteStudentProfile(ctx, userID); err != nil {
		return fmt.Errorf("delete student profile: %w", err)
	}
	// CASCADE on the FKs already wipes student_profile_programs rows, but
	// delete the programs explicitly so the operation is symmetric with
	// ReplaceProgramIDs and any orphan rows from earlier schema versions
	// get cleaned up too.
	if err := r.queries.DeleteStudentProfilePrograms(ctx, userID); err != nil {
		return fmt.Errorf("delete student profile programs: %w", err)
	}
	return nil
}

func (r *repository) ReplaceProgramIDs(ctx context.Context, userID string, ids []string) error {
	if err := r.queries.DeleteStudentProfilePrograms(ctx, userID); err != nil {
		return fmt.Errorf("clear student profile programs: %w", err)
	}
	for _, id := range ids {
		if err := r.queries.AddStudentProfileProgram(ctx, sqlc.AddStudentProfileProgramParams{
			UserID:    userID,
			ProgramID: id,
		}); err != nil {
			return fmt.Errorf("insert student profile program %s: %w", id, err)
		}
	}
	return nil
}

func (r *repository) ListProgramIDs(ctx context.Context, userID string) ([]string, error) {
	ids, err := r.queries.ListStudentProfileProgramIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list student profile program ids: %w", err)
	}
	return ids, nil
}

// ExistingProgramIDs returns the subset of ids that match a real program
// row, so the service can detect and reject dangling references with a
// 400 instead of letting an FK violation surface as a 500.
func (r *repository) ExistingProgramIDs(ctx context.Context, ids []string) (map[string]struct{}, error) {
	if len(ids) == 0 {
		return map[string]struct{}{}, nil
	}
	rows, err := r.queries.GetExistingProgramIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get existing program ids: %w", err)
	}
	set := make(map[string]struct{}, len(rows))
	for _, id := range rows {
		set[id] = struct{}{}
	}
	return set, nil
}

// listRecommendedSQL projects the same columns as favorites' listUniversitiesSQL
// so it can scan directly into university.UniversityListItem. Column order is
// explicit — schema.sql order doesn't match the runtime table order because of
// ALTER TABLE ADD COLUMN migrations, so SELECT * would mis-scan.
//
// Filters are conditionally appended based on which profile fields are set:
//   - always: status = 'published'
//   - country set: u.country = $N
//   - budget set: u.tuition_min <= $N (permissive — see service docs)
//   - program_ids set: EXISTS subquery against university_degree_levels
//     (reuses idx_university_degree_levels_degree_level_id)
const listRecommendedSQLTemplate = `
SELECT
    u.id,
    u.name,
    u.slug,
    COALESCE(u.country, '')                AS country,
    COALESCE(u.state, '')                  AS state,
    COALESCE(u.city, '')                   AS city,
    COALESCE(u.logo, '')                   AS logo,
    COALESCE(u.cover_image, '')            AS cover_image,
    COALESCE(u.institution_type, '')       AS institution_type,
    COALESCE(u.campus_setting, '')         AS campus_setting,
    COALESCE(u.tuition_min, 0)             AS tuition_min,
    COALESCE(u.tuition_max, 0)             AS tuition_max,
    COALESCE(u.acceptance_rate, 0)::float8 AS acceptance_rate,
    u.is_popular,
    u.is_featured,
    EXISTS (
        SELECT 1 FROM users rep
        WHERE rep.representative_university_id = u.id
    ) AS has_representative
FROM universities u
WHERE u.status = 'published'
%s%s%s
ORDER BY u.is_featured DESC, u.is_popular DESC, u.tuition_min ASC, u.name ASC
LIMIT $%d OFFSET $%d
`

// buildRecommendedWhere builds the dynamic WHERE fragment (each filter
// appended as " AND …", no leading space) and the matching positional
// args slice. The program_ids filter is added separately by the caller
// because programs live in a junction table that requires its own
// ListProgramIDs round-trip.
//
// The service layer guarantees at least one filter is set before
// calling here, but the resulting SQL is still correct with zero
// filters (it falls through to "status = 'published'").
func buildRecommendedWhere(p sqlc.StudentProfile) (string, []any) {
	var clauses []string
	var args []any

	if p.IntendedCountry != nil && *p.IntendedCountry != "" {
		args = append(args, *p.IntendedCountry)
		clauses = append(clauses, fmt.Sprintf(" AND u.country = $%d", len(args)))
	}
	if p.Budget != nil {
		args = append(args, *p.Budget)
		clauses = append(clauses, fmt.Sprintf(" AND u.tuition_min <= $%d", len(args)))
	}

	return strings.Join(clauses, ""), args
}

// buildRecommendedSQL returns the final SQL string and the full args slice
// for the recommendation list query. programIDs is the resolved list of
// program UUIDs the student has saved (may be empty).
func buildRecommendedSQL(profile sqlc.StudentProfile, programIDs []string, limit, offset int) (string, []any) {
	base, args := buildRecommendedWhere(profile)

	var programFragment string
	if len(programIDs) > 0 {
		args = append(args, programIDs)
		programFragment = fmt.Sprintf(
			" AND EXISTS (SELECT 1 FROM university_degree_levels udl WHERE udl.university_id = u.id AND udl.degree_level_id IN (SELECT degree_id FROM programs WHERE id = ANY($%d::uuid[])))",
			len(args),
		)
	}

	args = append(args, limit, offset)
	sql := fmt.Sprintf(listRecommendedSQLTemplate, base, programFragment, "", len(args)-1, len(args))
	return sql, args
}

// buildRecommendedCountSQL is the COUNT(*) companion for the WHERE
// fragment. Same filter logic, no ORDER BY/LIMIT, no columns.
func buildRecommendedCountSQL(profile sqlc.StudentProfile, programIDs []string) (string, []any) {
	where, args := buildRecommendedWhere(profile)
	var programFragment string
	if len(programIDs) > 0 {
		args = append(args, programIDs)
		programFragment = fmt.Sprintf(
			" AND EXISTS (SELECT 1 FROM university_degree_levels udl WHERE udl.university_id = u.id AND udl.degree_level_id IN (SELECT degree_id FROM programs WHERE id = ANY($%d::uuid[])))",
			len(args),
		)
	}
	countSQL := "SELECT COUNT(*) FROM universities u WHERE u.status = 'published'" + where + programFragment
	return countSQL, args
}

func (r *repository) ListRecommendedUniversities(ctx context.Context, profile sqlc.StudentProfile, programIDs []string, q pagination.Query) ([]university.UniversityListItem, int64, error) {
	countSQL, countArgs := buildRecommendedCountSQL(profile, programIDs)
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count recommended universities: %w", err)
	}

	listSQL, listArgs := buildRecommendedSQL(profile, programIDs, q.Limit(), q.Offset())
	rows, err := r.pool.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list recommended universities: %w", err)
	}
	defer rows.Close()

	items := []university.UniversityListItem{}
	for rows.Next() {
		var u university.UniversityListItem
		if err := rows.Scan(
			&u.ID, &u.Name, &u.Slug,
			&u.Country, &u.State, &u.City, &u.Logo,
			&u.CoverImage, &u.InstitutionType, &u.CampusSetting,
			&u.TuitionMin, &u.TuitionMax, &u.AcceptanceRate,
			&u.IsPopular, &u.IsFeatured, &u.HasRepresentative,
		); err != nil {
			return nil, 0, fmt.Errorf("scan recommended university: %w", err)
		}
		items = append(items, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate recommended universities: %w", err)
	}
	return items, total, nil
}