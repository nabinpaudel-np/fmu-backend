package university

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
	Get(ctx context.Context, q pagination.Query, f Filters) ([]sqlc.University, int64, error)
	GetByID(ctx context.Context, id string) (sqlc.University, error)
	Search(ctx context.Context, q string) ([]sqlc.SearchUniversitiesRow, error)
	GetUniversityDegreeLevels(ctx context.Context, universityID string) ([]sqlc.DegreeLevel, error)
	GetUniversityMajors(ctx context.Context, universityID string) ([]sqlc.Major, error)
	GetUniversityStudyFormats(ctx context.Context, universityID string) ([]sqlc.StudyFormat, error)
	GetUniversitySpecialAffiliations(ctx context.Context, universityID string) ([]sqlc.SpecialAffiliation, error)
	GetUniversityAthletics(ctx context.Context, universityID string) ([]sqlc.Athletic, error)
	GetUniversitySupportServices(ctx context.Context, universityID string) ([]sqlc.SupportService, error)
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

// maxSearchResults caps the payload for the /universities/search dropdown —
// search has no filters to combine, so a hard cap keeps responses snappy.
const maxSearchResults = 50

func NewUniversityRepository(queries *sqlc.Queries, pool *pgxpool.Pool) UniversityRepository {
	return &universityRepository{queries: queries, pool: pool}
}

func (r *universityRepository) Search(ctx context.Context, q string) ([]sqlc.SearchUniversitiesRow, error) {
	return r.queries.SearchUniversities(ctx, sqlc.SearchUniversitiesParams{
		Similarity: q,
		Limit:      int32(maxSearchResults),
	})
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

func (r *universityRepository) Get(ctx context.Context, q pagination.Query, f Filters) ([]sqlc.University, int64, error) {
	where, args := buildUniversitiesWhere(f)

	var total int64
	countSQL := "SELECT COUNT(*) FROM universities u WHERE 1=1" + where
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count universities: %w", err)
	}

	listArgs := append(append([]any{}, args...), q.Limit(), q.Offset())
	// Column list spelled out: ALTER TABLE ADD COLUMN appends new columns to the
	// runtime table, but schema.sql declares them inline — so SELECT * doesn't
	// match the sqlc struct scan order.
	listSQL := fmt.Sprintf(
		"SELECT id, name, slug, overview, excerpt, country, state, city, full_location, cover_image, logo, institution_type, campus_setting, in_state_tuition, out_of_state_tuition, international_tuition, need_based_aid, merit_scholarships, work_study, no_application_fee, acceptance_rate, testing_policy, sat_range, act_range, on_campus_housing, freshmen_required_on_campus, contact_email, contact_phone, website, zipcode, tuition_min, tuition_max, avg_high_school_gpa, founded_year, campus_size, gallery_images, is_popular, is_featured, created_at, updated_at FROM universities u WHERE 1=1%s ORDER BY u.name LIMIT $%d OFFSET $%d",
		where, len(listArgs)-1, len(listArgs),
	)

	rows, err := r.pool.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list universities: %w", err)
	}
	defer rows.Close()

	unis, err := collectUniversities(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("scan universities: %w", err)
	}
	return unis, total, nil
}

func collectUniversities(rows pgx.Rows) ([]sqlc.University, error) {
	items := []sqlc.University{}
	for rows.Next() {
		var u sqlc.University
		if err := rows.Scan(
			&u.ID, &u.Name, &u.Slug, &u.Overview, &u.Excerpt,
			&u.Country, &u.State, &u.City, &u.FullLocation,
			&u.CoverImage, &u.Logo,
			&u.InstitutionType, &u.CampusSetting,
			&u.InStateTuition, &u.OutOfStateTuition, &u.InternationalTuition,
			&u.NeedBasedAid, &u.MeritScholarships, &u.WorkStudy, &u.NoApplicationFee,
			&u.AcceptanceRate, &u.TestingPolicy, &u.SatRange, &u.ActRange,
			&u.OnCampusHousing, &u.FreshmenRequiredOnCampus,
			&u.ContactEmail, &u.ContactPhone, &u.Website,
			&u.Zipcode, &u.TuitionMin, &u.TuitionMax, &u.AvgHighSchoolGpa,
			&u.FoundedYear, &u.CampusSize, &u.GalleryImages,
			&u.IsPopular, &u.IsFeatured,
			&u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, u)
	}
	return items, rows.Err()
}

// buildUniversitiesWhere returns a parameterized WHERE fragment (leading " AND ")
// plus the matching args slice. Empty Filters produces empty output.
func buildUniversitiesWhere(f Filters) (string, []any) {
	var clauses []string
	var args []any

	eq := func(format, val string) {
		if val == "" {
			return
		}
		args = append(args, val)
		clauses = append(clauses, fmt.Sprintf(format, len(args)))
	}

	// EXISTS over a join table — one subquery per multi-value filter so they
	// resolve to a single index hit instead of repeated joins.
	addExists := func(joinTable, lookupTable, idColumn string, values []string) {
		if len(values) == 0 {
			return
		}
		args = append(args, values)
		clauses = append(clauses, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM %s ul JOIN %s l ON l.id = ul.%s WHERE ul.university_id = u.id AND l.name = ANY($%d::text[]))",
			joinTable, lookupTable, idColumn, len(args),
		))
	}

	eq("u.institution_type = $%d", f.InstitutionType)
	eq("u.testing_policy = $%d", f.TestingPolicy)
	eq("u.country = $%d", f.Country)
	eq("u.state = $%d", f.State)
	eq("u.city = $%d", f.City)

	if len(f.CampusSettings) > 0 {
		args = append(args, f.CampusSettings)
		clauses = append(clauses, fmt.Sprintf("u.campus_setting = ANY($%d::text[])", len(args)))
	}

	if f.TuitionMin != nil {
		args = append(args, *f.TuitionMin)
		clauses = append(clauses, fmt.Sprintf("u.tuition_min >= $%d", len(args)))
	}
	if f.TuitionMax != nil {
		args = append(args, *f.TuitionMax)
		clauses = append(clauses, fmt.Sprintf("u.tuition_max <= $%d", len(args)))
	}
	if f.AcceptanceMin != nil {
		args = append(args, *f.AcceptanceMin)
		clauses = append(clauses, fmt.Sprintf("u.acceptance_rate >= $%d", len(args)))
	}
	if f.AcceptanceMax != nil {
		args = append(args, *f.AcceptanceMax)
		clauses = append(clauses, fmt.Sprintf("u.acceptance_rate <= $%d", len(args)))
	}

	if v := f.NeedBasedAid; v != nil {
		args = append(args, *v)
		clauses = append(clauses, fmt.Sprintf("u.need_based_aid = $%d", len(args)))
	}
	if v := f.MeritScholarships; v != nil {
		args = append(args, *v)
		clauses = append(clauses, fmt.Sprintf("u.merit_scholarships = $%d", len(args)))
	}
	if v := f.NoApplicationFee; v != nil {
		args = append(args, *v)
		clauses = append(clauses, fmt.Sprintf("u.no_application_fee = $%d", len(args)))
	}
	if v := f.OnCampusHousing; v != nil {
		args = append(args, *v)
		clauses = append(clauses, fmt.Sprintf("u.on_campus_housing = $%d", len(args)))
	}

	addExists("university_majors", "majors", "major_id", f.Majors)
	addExists("university_degree_levels", "degree_levels", "degree_level_id", f.DegreeLevels)
	addExists("university_study_formats", "study_formats", "study_format_id", f.StudyFormats)
	addExists("university_special_affiliations", "special_affiliations", "special_affiliation_id", f.SpecialAffiliations)
	addExists("university_athletics", "athletics", "athletic_id", f.Athletics)

	// Multiple has_X=true AND together; within one lookup param it's ANY (handled
	// by addExists above).
	for _, name := range sortedSupportServiceNames(f.HasSupportService) {
		args = append(args, name)
		clauses = append(clauses, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM university_support_services uss JOIN support_services ss ON ss.id = uss.support_service_id WHERE uss.university_id = u.id AND ss.name = $%d)",
			len(args),
		))
	}

	if len(clauses) == 0 {
		return "", nil
	}
	return " AND " + strings.Join(clauses, " AND "), args
}

// Sorted: map iteration is randomized, and we want identical Filters to
// produce identical parameter order so Postgres reuses a prepared plan.
func sortedSupportServiceNames(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

func (r *universityRepository) GetByID(ctx context.Context, id string) (sqlc.University, error) {
	return r.queries.GetUniversityByID(ctx, id)
}

func (r *universityRepository) GetUniversityDegreeLevels(ctx context.Context, universityID string) ([]sqlc.DegreeLevel, error) {
	return r.queries.GetUniversityDegreeLevels(ctx, universityID)
}

func (r *universityRepository) GetUniversityMajors(ctx context.Context, universityID string) ([]sqlc.Major, error) {
	return r.queries.GetUniversityMajors(ctx, universityID)
}

func (r *universityRepository) GetUniversityStudyFormats(ctx context.Context, universityID string) ([]sqlc.StudyFormat, error) {
	return r.queries.GetUniversityStudyFormats(ctx, universityID)
}

func (r *universityRepository) GetUniversitySpecialAffiliations(ctx context.Context, universityID string) ([]sqlc.SpecialAffiliation, error) {
	return r.queries.GetUniversitySpecialAffiliations(ctx, universityID)
}

func (r *universityRepository) GetUniversityAthletics(ctx context.Context, universityID string) ([]sqlc.Athletic, error) {
	return r.queries.GetUniversityAthletics(ctx, universityID)
}

func (r *universityRepository) GetUniversitySupportServices(ctx context.Context, universityID string) ([]sqlc.SupportService, error) {
	return r.queries.GetUniversitySupportServices(ctx, universityID)
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
