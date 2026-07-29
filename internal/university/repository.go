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
	Patch(ctx context.Context, id string, req *PatchUniversityRequest) (sqlc.University, error)
	Get(ctx context.Context, q pagination.Query, f Filters) ([]sqlc.University, int64, error)
	GetByID(ctx context.Context, id string) (sqlc.University, error)
	Search(ctx context.Context, q string) ([]sqlc.SearchUniversitiesRow, error)
	RepresentedIDs(ctx context.Context, ids []string) (map[string]struct{}, error)
	Stats(ctx context.Context) (*UniversityStats, error)
	Publish(ctx context.Context, id string) (sqlc.University, error)
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

type UniversityStats struct {
	TotalUniversities int64
	TotalCountries    int64
	TotalFeatured     int64
	TotalPopular      int64
}

type universityRepository struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
}

// maxSearchResults caps the /universities/search dropdown. No filters to
// narrow with, so a hard cap keeps responses snappy.
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

func (r *universityRepository) RepresentedIDs(ctx context.Context, ids []string) (map[string]struct{}, error) {
	if len(ids) == 0 {
		return map[string]struct{}{}, nil
	}
	rows, err := r.queries.ListRepresentedUniversityIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list represented university ids: %w", err)
	}
	set := make(map[string]struct{}, len(rows))
	for _, id := range rows {
		set[id] = struct{}{}
	}
	return set, nil
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
	// Columns spelled out: schema.sql order ≠ runtime table order, so
	// SELECT * doesn't match the sqlc struct scan order.
	listSQL := fmt.Sprintf(
		"SELECT id, name, slug, overview, excerpt, country, state, city, full_location, cover_image, logo, institution_type, campus_setting, in_state_tuition, out_of_state_tuition, international_tuition, need_based_aid, merit_scholarships, work_study, no_application_fee, acceptance_rate, testing_policy, sat_range, act_range, on_campus_housing, freshmen_required_on_campus, contact_email, contact_phone, website, zipcode, tuition_min, tuition_max, avg_high_school_gpa, founded_year, campus_size, gallery_images, is_popular, is_featured, maps_url, full_address, employment_rate, research_output, housing_type, seo_title, seo_description, status, published_at, created_at, updated_at FROM universities u WHERE 1=1%s ORDER BY u.name LIMIT $%d OFFSET $%d",
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
			&u.MapsUrl, &u.FullAddress, &u.EmploymentRate, &u.ResearchOutput, &u.HousingType,
			&u.SeoTitle, &u.SeoDescription,
			&u.Status, &u.PublishedAt,
			&u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, u)
	}
	return items, rows.Err()
}

// buildUniversitiesWhere returns a parameterized WHERE fragment (with a
// leading " AND ") and the matching args. Empty Filters produces empty output.
func buildUniversitiesWhere(f Filters) (string, []any) {
	var orClauses []string
	var andClauses []string
	var args []any

	// Status is always AND-ed (it scopes the whole query). The rest of the
	// filters are OR-ed because they come from optional multi-select facets:
	// "find unis matching any of these criteria". The handler validates that
	// non-admin callers can only pass "published".
	if f.Status != "" {
		args = append(args, f.Status)
		andClauses = append(andClauses, fmt.Sprintf("u.status = $%d", len(args)))
	}
	if v := f.IsPopular; v != nil {
		args = append(args, *v)
		andClauses = append(andClauses, fmt.Sprintf("u.is_popular = $%d", len(args)))
	}
	if v := f.IsFeatured; v != nil {
		args = append(args, *v)
		andClauses = append(andClauses, fmt.Sprintf("u.is_featured = $%d", len(args)))
	}

	eq := func(format, val string) {
		if val == "" {
			return
		}
		args = append(args, val)
		orClauses = append(orClauses, fmt.Sprintf(format, len(args)))
	}

	// EXISTS subquery per filter — single index hit beats a join per row.
	addExists := func(joinTable, lookupTable, idColumn string, values []string) {
		if len(values) == 0 {
			return
		}
		args = append(args, values)
		orClauses = append(orClauses, fmt.Sprintf(
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
		orClauses = append(orClauses, fmt.Sprintf("u.campus_setting = ANY($%d::text[])", len(args)))
	}

	if f.TuitionMin != nil {
		args = append(args, *f.TuitionMin)
		orClauses = append(orClauses, fmt.Sprintf("u.tuition_min >= $%d", len(args)))
	}
	if f.TuitionMax != nil {
		args = append(args, *f.TuitionMax)
		orClauses = append(orClauses, fmt.Sprintf("u.tuition_max <= $%d", len(args)))
	}
	if f.AcceptanceMin != nil {
		args = append(args, *f.AcceptanceMin)
		orClauses = append(orClauses, fmt.Sprintf("u.acceptance_rate >= $%d", len(args)))
	}
	if f.AcceptanceMax != nil {
		args = append(args, *f.AcceptanceMax)
		orClauses = append(orClauses, fmt.Sprintf("u.acceptance_rate <= $%d", len(args)))
	}

	if v := f.NeedBasedAid; v != nil {
		args = append(args, *v)
		orClauses = append(orClauses, fmt.Sprintf("u.need_based_aid = $%d", len(args)))
	}
	if v := f.MeritScholarships; v != nil {
		args = append(args, *v)
		orClauses = append(orClauses, fmt.Sprintf("u.merit_scholarships = $%d", len(args)))
	}
	if v := f.NoApplicationFee; v != nil {
		args = append(args, *v)
		orClauses = append(orClauses, fmt.Sprintf("u.no_application_fee = $%d", len(args)))
	}
	if v := f.OnCampusHousing; v != nil {
		args = append(args, *v)
		orClauses = append(orClauses, fmt.Sprintf("u.on_campus_housing = $%d", len(args)))
	}

	addExists("university_majors", "majors", "major_id", f.Majors)
	addExists("university_degree_levels", "degree_levels", "degree_level_id", f.DegreeLevels)
	addExists("university_study_formats", "study_formats", "study_format_id", f.StudyFormats)
	addExists("university_special_affiliations", "special_affiliations", "special_affiliation_id", f.SpecialAffiliations)
	addExists("university_athletics", "athletics", "athletic_id", f.Athletics)

	// All filter clauses are OR-ed: a row matches if any one filter is true.
	// Within a multi-value lookup param, addExists still uses ANY (OR over
	// values); each HasSupportService key also becomes its own EXISTS clause
	// here, so multi-select services also OR (any-of).
	for _, name := range sortedSupportServiceNames(f.HasSupportService) {
		args = append(args, name)
		orClauses = append(orClauses, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM university_support_services uss JOIN support_services ss ON ss.id = uss.support_service_id WHERE uss.university_id = u.id AND ss.name = $%d)",
			len(args),
		))
	}

	parts := append([]string{}, andClauses...)
	if len(orClauses) > 0 {
		parts = append(parts, "("+strings.Join(orClauses, " OR ")+")")
	}
	if len(parts) == 0 {
		return "", nil
	}
	return " AND " + strings.Join(parts, " AND "), args
}

// Sorted: identical Filters must produce identical param order so
// Postgres can reuse a prepared plan.
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

func (r *universityRepository) Patch(ctx context.Context, id string, req *PatchUniversityRequest) (sqlc.University, error) {
	var err error
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

	missing := map[string][]string{}
	collectMissing := func(table string, ids []string, getExisting func(context.Context, []string) ([]string, error)) {
		if len(ids) == 0 {
			return
		}
		existing, err := getExisting(ctx, ids)
		if err != nil {
			return
		}
		if m := findMissing(existing, ids); len(m) > 0 {
			missing[table] = m
		}
	}

	if req.DegreeLevelIDs != nil {
		collectMissing("degree_levels", *req.DegreeLevelIDs, q.GetExistingDegreeLevelIDs)
	}
	if req.MajorIDs != nil {
		collectMissing("majors", *req.MajorIDs, q.GetExistingMajorIDs)
	}
	if req.StudyFormatIDs != nil {
		collectMissing("study_formats", *req.StudyFormatIDs, q.GetExistingStudyFormatIDs)
	}
	if req.SpecialAffiliationIDs != nil {
		collectMissing("special_affiliations", *req.SpecialAffiliationIDs, q.GetExistingSpecialAffiliationIDs)
	}
	if req.AthleticIDs != nil {
		collectMissing("athletics", *req.AthleticIDs, q.GetExistingAthleticIDs)
	}
	if req.SupportServiceIDs != nil {
		collectMissing("support_services", *req.SupportServiceIDs, q.GetExistingSupportServiceIDs)
	}

	if len(missing) > 0 {
		return sqlc.University{}, &errs.InvalidReferencesError{References: missing}
	}

	sets := []string{}
	args := []any{}

	addSet := func(col string, val any) {
		args = append(args, val)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}

	if req.Name != nil {
		addSet("name", *req.Name)
	}
	if req.Slug != nil {
		addSet("slug", *req.Slug)
	}
	if req.Overview != nil {
		addSet("overview", *req.Overview)
	}
	if req.Excerpt != nil {
		addSet("excerpt", *req.Excerpt)
	}
	if req.Country != nil {
		addSet("country", *req.Country)
	}
	if req.State != nil {
		addSet("state", *req.State)
	}
	if req.City != nil {
		addSet("city", *req.City)
	}
	if req.FullLocation != nil {
		addSet("full_location", *req.FullLocation)
	}
	if req.CoverImage != nil {
		addSet("cover_image", *req.CoverImage)
	}
	if req.Logo != nil {
		addSet("logo", *req.Logo)
	}
	if req.InstitutionType != nil {
		addSet("institution_type", *req.InstitutionType)
	}
	if req.CampusSetting != nil {
		addSet("campus_setting", *req.CampusSetting)
	}
	if req.InStateTuition != nil {
		addSet("in_state_tuition", nullableNumeric(*req.InStateTuition))
	}
	if req.OutOfStateTuition != nil {
		addSet("out_of_state_tuition", nullableNumeric(*req.OutOfStateTuition))
	}
	if req.InternationalTuition != nil {
		addSet("international_tuition", nullableNumeric(*req.InternationalTuition))
	}
	if req.NeedBasedAid != nil {
		addSet("need_based_aid", *req.NeedBasedAid)
	}
	if req.MeritScholarships != nil {
		addSet("merit_scholarships", *req.MeritScholarships)
	}
	if req.WorkStudy != nil {
		addSet("work_study", *req.WorkStudy)
	}
	if req.NoApplicationFee != nil {
		addSet("no_application_fee", *req.NoApplicationFee)
	}
	if req.AcceptanceRate != nil {
		addSet("acceptance_rate", nullableNumeric(*req.AcceptanceRate))
	}
	if req.TestingPolicy != nil {
		addSet("testing_policy", *req.TestingPolicy)
	}
	if req.SatRange != nil {
		addSet("sat_range", *req.SatRange)
	}
	if req.ActRange != nil {
		addSet("act_range", *req.ActRange)
	}
	if req.OnCampusHousing != nil {
		addSet("on_campus_housing", *req.OnCampusHousing)
	}
	if req.FreshmenRequiredOnCampus != nil {
		addSet("freshmen_required_on_campus", *req.FreshmenRequiredOnCampus)
	}
	if req.ContactEmail != nil {
		addSet("contact_email", *req.ContactEmail)
	}
	if req.ContactPhone != nil {
		addSet("contact_phone", *req.ContactPhone)
	}
	if req.Website != nil {
		addSet("website", *req.Website)
	}
	if req.Zipcode != nil {
		addSet("zipcode", *req.Zipcode)
	}
	if req.TuitionMin != nil {
		addSet("tuition_min", *req.TuitionMin)
	}
	if req.TuitionMax != nil {
		addSet("tuition_max", *req.TuitionMax)
	}
	if req.AvgHighSchoolGpa != nil {
		addSet("avg_high_school_gpa", nullableNumeric(*req.AvgHighSchoolGpa))
	}
	if req.FoundedYear != nil {
		addSet("founded_year", int16(*req.FoundedYear))
	}
	if req.CampusSize != nil {
		addSet("campus_size", *req.CampusSize)
	}
	if req.GalleryImages != nil {
		addSet("gallery_images", *req.GalleryImages)
	}
	if req.IsPopular != nil {
		addSet("is_popular", *req.IsPopular)
	}
	if req.IsFeatured != nil {
		addSet("is_featured", *req.IsFeatured)
	}
	if req.MapsUrl != nil {
		addSet("maps_url", *req.MapsUrl)
	}
	if req.FullAddress != nil {
		addSet("full_address", *req.FullAddress)
	}
	if req.EmploymentRate != nil {
		addSet("employment_rate", nullableNumeric(*req.EmploymentRate))
	}
	if req.ResearchOutput != nil {
		addSet("research_output", *req.ResearchOutput)
	}
	if req.HousingType != nil {
		addSet("housing_type", *req.HousingType)
	}
	if req.SeoTitle != nil {
		addSet("seo_title", *req.SeoTitle)
	}
	if req.SeoDescription != nil {
		addSet("seo_description", *req.SeoDescription)
	}

	sets = append(sets, "updated_at = now()")
	args = append(args, id)
	// RETURNING lists columns in scan order, not SELECT *. The ALTER TABLE ADD
	// COLUMN migration appended fields after created_at/updated_at, so the
	// physical column order no longer matches the schema.sql declaration.
	sql := fmt.Sprintf("UPDATE universities SET %s WHERE id = $%d RETURNING id, name, slug, overview, excerpt, country, state, city, full_location, cover_image, logo, institution_type, campus_setting, in_state_tuition, out_of_state_tuition, international_tuition, need_based_aid, merit_scholarships, work_study, no_application_fee, acceptance_rate, testing_policy, sat_range, act_range, on_campus_housing, freshmen_required_on_campus, contact_email, contact_phone, website, zipcode, tuition_min, tuition_max, avg_high_school_gpa, founded_year, campus_size, gallery_images, is_popular, is_featured, maps_url, full_address, employment_rate, research_output, housing_type, seo_title, seo_description, status, published_at, created_at, updated_at",
		strings.Join(sets, ", "), len(args))

	var row sqlc.University
	err = tx.QueryRow(ctx, sql, args...).Scan(
		&row.ID, &row.Name, &row.Slug, &row.Overview, &row.Excerpt,
		&row.Country, &row.State, &row.City, &row.FullLocation,
		&row.CoverImage, &row.Logo,
		&row.InstitutionType, &row.CampusSetting,
		&row.InStateTuition, &row.OutOfStateTuition, &row.InternationalTuition,
		&row.NeedBasedAid, &row.MeritScholarships, &row.WorkStudy, &row.NoApplicationFee,
		&row.AcceptanceRate, &row.TestingPolicy, &row.SatRange, &row.ActRange,
		&row.OnCampusHousing, &row.FreshmenRequiredOnCampus,
		&row.ContactEmail, &row.ContactPhone, &row.Website,
		&row.Zipcode, &row.TuitionMin, &row.TuitionMax, &row.AvgHighSchoolGpa,
		&row.FoundedYear, &row.CampusSize, &row.GalleryImages,
		&row.IsPopular, &row.IsFeatured,
		&row.MapsUrl, &row.FullAddress, &row.EmploymentRate, &row.ResearchOutput, &row.HousingType,
		&row.SeoTitle, &row.SeoDescription,
		&row.Status, &row.PublishedAt,
		&row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.University{}, errs.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && req.Slug != nil {
			return sqlc.University{}, fmt.Errorf("%w (slug=%s)", errs.ErrUniversitySlugTaken, *req.Slug)
		}
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return sqlc.University{}, errs.ErrNotFound
		}
		return sqlc.University{}, err
	}

	replaceLookup := func(ids []string, deleteFn func(context.Context, string) error, insertFn func(context.Context, []string) error) error {
		if err := deleteFn(ctx, id); err != nil {
			return err
		}
		if len(ids) > 0 {
			if err := insertFn(ctx, ids); err != nil {
				return err
			}
		}
		return nil
	}

	if req.DegreeLevelIDs != nil {
		err = replaceLookup(*req.DegreeLevelIDs, q.DeleteUniversityDegreeLevels, func(ctx context.Context, ids []string) error {
			return q.InsertUniversityDegreeLevels(ctx, sqlc.InsertUniversityDegreeLevelsParams{
				UniversityID: row.ID,
				Column2:      ids,
			})
		})
		if err != nil {
			return sqlc.University{}, err
		}
	}
	if req.MajorIDs != nil {
		err = replaceLookup(*req.MajorIDs, q.DeleteUniversityMajors, func(ctx context.Context, ids []string) error {
			return q.InsertUniversityMajors(ctx, sqlc.InsertUniversityMajorsParams{
				UniversityID: row.ID,
				Column2:      ids,
			})
		})
		if err != nil {
			return sqlc.University{}, err
		}
	}
	if req.StudyFormatIDs != nil {
		err = replaceLookup(*req.StudyFormatIDs, q.DeleteUniversityStudyFormats, func(ctx context.Context, ids []string) error {
			return q.InsertUniversityStudyFormats(ctx, sqlc.InsertUniversityStudyFormatsParams{
				UniversityID: row.ID,
				Column2:      ids,
			})
		})
		if err != nil {
			return sqlc.University{}, err
		}
	}
	if req.SpecialAffiliationIDs != nil {
		err = replaceLookup(*req.SpecialAffiliationIDs, q.DeleteUniversitySpecialAffiliations, func(ctx context.Context, ids []string) error {
			return q.InsertUniversitySpecialAffiliations(ctx, sqlc.InsertUniversitySpecialAffiliationsParams{
				UniversityID: row.ID,
				Column2:      ids,
			})
		})
		if err != nil {
			return sqlc.University{}, err
		}
	}
	if req.AthleticIDs != nil {
		err = replaceLookup(*req.AthleticIDs, q.DeleteUniversityAthletics, func(ctx context.Context, ids []string) error {
			return q.InsertUniversityAthletics(ctx, sqlc.InsertUniversityAthleticsParams{
				UniversityID: row.ID,
				Column2:      ids,
			})
		})
		if err != nil {
			return sqlc.University{}, err
		}
	}
	if req.SupportServiceIDs != nil {
		err = replaceLookup(*req.SupportServiceIDs, q.DeleteUniversitySupportServices, func(ctx context.Context, ids []string) error {
			return q.InsertUniversitySupportServices(ctx, sqlc.InsertUniversitySupportServicesParams{
				UniversityID: row.ID,
				Column2:      ids,
			})
		})
		if err != nil {
			return sqlc.University{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return sqlc.University{}, err
	}

	return row, nil
}

func (r *universityRepository) Stats(ctx context.Context) (*UniversityStats, error) {
	const query = `
		SELECT
			COUNT(*)::bigint AS total_universities,
			COUNT(DISTINCT country)::bigint AS total_countries,
			COUNT(*) FILTER (WHERE is_featured)::bigint AS total_featured,
			COUNT(*) FILTER (WHERE is_popular)::bigint AS total_popular
		FROM universities
		WHERE status = 'published'
	`
	var s UniversityStats
	if err := r.pool.QueryRow(ctx, query).Scan(
		&s.TotalUniversities,
		&s.TotalCountries,
		&s.TotalFeatured,
		&s.TotalPopular,
	); err != nil {
		return nil, fmt.Errorf("stats: %w", err)
	}
	return &s, nil
}

func (r *universityRepository) Publish(ctx context.Context, id string) (sqlc.University, error) {
	row, err := r.queries.PublishUniversity(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.University{}, errs.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return sqlc.University{}, errs.ErrNotFound
		}
		return sqlc.University{}, err
	}
	return row, nil
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
