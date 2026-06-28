package university

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/errs"
)

type lookupIDs struct {
	DegreeLevelIDs        []string
	MajorIDs              []string
	StudyFormatIDs        []string
	SpecialAffiliationIDs []string
	AthleticIDs           []string
	SupportServiceIDs     []string
}

type UniversityRepository interface {
	Create(ctx context.Context, params sqlc.CreateUniversityParams, ids lookupIDs) (sqlc.University, error)
	GetMajors(ctx context.Context) ([]sqlc.Major, error)
	GetDegreeLevels(ctx context.Context) ([]sqlc.DegreeLevel, error)
	GetStudyFormats(ctx context.Context) ([]sqlc.StudyFormat, error)
	GetSpecialAffiliations(ctx context.Context) ([]sqlc.SpecialAffiliation, error)
	GetAthletics(ctx context.Context) ([]sqlc.Athletic, error)
	GetSupportServices(ctx context.Context) ([]sqlc.SupportService, error)
}

type universityRepository struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
}

func NewUniversityRepository(queries *sqlc.Queries, pool *pgxpool.Pool) UniversityRepository {
	return &universityRepository{queries: queries, pool: pool}
}

func (r *universityRepository) Create(ctx context.Context, params sqlc.CreateUniversityParams, ids lookupIDs) (sqlc.University, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return sqlc.University{}, err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	q := r.queries.WithTx(tx)

	missing, err := validateReferences(ctx, q, ids)
	if err != nil {
		return sqlc.University{}, err
	}
	if len(missing) > 0 {
		return sqlc.University{}, &errs.InvalidReferencesError{References: missing}
	}

	row, err := q.CreateUniversity(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return sqlc.University{}, fmt.Errorf("%w (slug=%s)", errs.ErrUniversitySlugTaken, params.Slug)
		}
		return sqlc.University{}, err
	}

	if len(ids.DegreeLevelIDs) > 0 {
		if err = q.InsertUniversityDegreeLevels(ctx, sqlc.InsertUniversityDegreeLevelsParams{
			UniversityID: row.ID,
			Column2:      ids.DegreeLevelIDs,
		}); err != nil {
			return sqlc.University{}, err
		}
	}
	if len(ids.MajorIDs) > 0 {
		if err = q.InsertUniversityMajors(ctx, sqlc.InsertUniversityMajorsParams{
			UniversityID: row.ID,
			Column2:      ids.MajorIDs,
		}); err != nil {
			return sqlc.University{}, err
		}
	}
	if len(ids.StudyFormatIDs) > 0 {
		if err = q.InsertUniversityStudyFormats(ctx, sqlc.InsertUniversityStudyFormatsParams{
			UniversityID: row.ID,
			Column2:      ids.StudyFormatIDs,
		}); err != nil {
			return sqlc.University{}, err
		}
	}
	if len(ids.SpecialAffiliationIDs) > 0 {
		if err = q.InsertUniversitySpecialAffiliations(ctx, sqlc.InsertUniversitySpecialAffiliationsParams{
			UniversityID: row.ID,
			Column2:      ids.SpecialAffiliationIDs,
		}); err != nil {
			return sqlc.University{}, err
		}
	}
	if len(ids.AthleticIDs) > 0 {
		if err = q.InsertUniversityAthletics(ctx, sqlc.InsertUniversityAthleticsParams{
			UniversityID: row.ID,
			Column2:      ids.AthleticIDs,
		}); err != nil {
			return sqlc.University{}, err
		}
	}
	if len(ids.SupportServiceIDs) > 0 {
		if err = q.InsertUniversitySupportServices(ctx, sqlc.InsertUniversitySupportServicesParams{
			UniversityID: row.ID,
			Column2:      ids.SupportServiceIDs,
		}); err != nil {
			return sqlc.University{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return sqlc.University{}, err
	}

	return row, nil
}

func (r *universityRepository) GetMajors(ctx context.Context) ([]sqlc.Major, error) {
	return r.queries.GetMajors(ctx)
}

func (r *universityRepository) GetDegreeLevels(ctx context.Context) ([]sqlc.DegreeLevel, error) {
	return r.queries.GetDegreeLevels(ctx)
}

func (r *universityRepository) GetStudyFormats(ctx context.Context) ([]sqlc.StudyFormat, error) {
	return r.queries.GetStudyFormats(ctx)
}

func (r *universityRepository) GetSpecialAffiliations(ctx context.Context) ([]sqlc.SpecialAffiliation, error) {
	return r.queries.GetSpecialAffiliations(ctx)
}

func (r *universityRepository) GetAthletics(ctx context.Context) ([]sqlc.Athletic, error) {
	return r.queries.GetAthletics(ctx)
}

func (r *universityRepository) GetSupportServices(ctx context.Context) ([]sqlc.SupportService, error) {
	return r.queries.GetSupportServices(ctx)
}

func validateReferences(ctx context.Context, q *sqlc.Queries, ids lookupIDs) (map[string][]string, error) {
	var missing map[string][]string

	record := func(table string, existing, requested []string) {
		if m := findMissing(existing, requested); len(m) > 0 {
			if missing == nil {
				missing = make(map[string][]string)
			}
			missing[table] = m
		}
	}

	existing, err := q.GetExistingDegreeLevelIDs(ctx, ids.DegreeLevelIDs)
	if err != nil {
		return nil, err
	}
	record("degree_levels", existing, ids.DegreeLevelIDs)

	existing, err = q.GetExistingMajorIDs(ctx, ids.MajorIDs)
	if err != nil {
		return nil, err
	}
	record("majors", existing, ids.MajorIDs)

	existing, err = q.GetExistingStudyFormatIDs(ctx, ids.StudyFormatIDs)
	if err != nil {
		return nil, err
	}
	record("study_formats", existing, ids.StudyFormatIDs)

	existing, err = q.GetExistingSpecialAffiliationIDs(ctx, ids.SpecialAffiliationIDs)
	if err != nil {
		return nil, err
	}
	record("special_affiliations", existing, ids.SpecialAffiliationIDs)

	existing, err = q.GetExistingAthleticIDs(ctx, ids.AthleticIDs)
	if err != nil {
		return nil, err
	}
	record("athletics", existing, ids.AthleticIDs)

	existing, err = q.GetExistingSupportServiceIDs(ctx, ids.SupportServiceIDs)
	if err != nil {
		return nil, err
	}
	record("support_services", existing, ids.SupportServiceIDs)

	return missing, nil
}

func findMissing(existing, requested []string) []string {
	found := make(map[string]struct{}, len(existing))
	for _, id := range existing {
		found[id] = struct{}{}
	}
	var missing []string
	for _, id := range requested {
		if _, ok := found[id]; !ok {
			missing = append(missing, id)
		}
	}
	return missing
}
