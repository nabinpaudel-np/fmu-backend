package scholarship

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
	EducationLevelIDs []string
	MajorIDs          []string
	DemographicIDs    []string
}

type ScholarshipRepository interface {
	Create(ctx context.Context, params sqlc.CreateScholarshipParams, ids lookupIDs) (sqlc.Scholarship, error)
	Patch(ctx context.Context, id string, req *UpdateScholarshipRequest) (sqlc.Scholarship, error)
	Get(ctx context.Context, q pagination.Query, f Filters) ([]ScholarshipListItem, int64, error)
	GetByID(ctx context.Context, id string) (sqlc.Scholarship, error)
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, q string) ([]sqlc.SearchScholarshipsRow, error)
	Publish(ctx context.Context, id string) (sqlc.Scholarship, error)

	GetScholarshipEducationLevels(ctx context.Context, scholarshipID string) ([]sqlc.EducationLevel, error)
	GetScholarshipMajors(ctx context.Context, scholarshipID string) ([]sqlc.Major, error)
	GetScholarshipDemographics(ctx context.Context, scholarshipID string) ([]sqlc.Demographic, error)

	GetEducationLevels(ctx context.Context) ([]sqlc.EducationLevel, error)
	GetDemographics(ctx context.Context) ([]sqlc.Demographic, error)
	GetMajors(ctx context.Context) ([]sqlc.Major, error)
}

type scholarshipRepository struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
}

// maxSearchResults caps the /scholarships/search dropdown.
const maxSearchResults = 50

func NewScholarshipRepository(queries *sqlc.Queries, pool *pgxpool.Pool) ScholarshipRepository {
	return &scholarshipRepository{queries: queries, pool: pool}
}

func (r *scholarshipRepository) Create(ctx context.Context, params sqlc.CreateScholarshipParams, ids lookupIDs) (sqlc.Scholarship, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return sqlc.Scholarship{}, err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	q := r.queries.WithTx(tx)

	missing, err := validateReferences(ctx, q, ids)
	if err != nil {
		return sqlc.Scholarship{}, err
	}
	if len(missing) > 0 {
		return sqlc.Scholarship{}, &errs.InvalidReferencesError{References: missing}
	}

	row, err := q.CreateScholarship(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				return sqlc.Scholarship{}, fmt.Errorf("%w (slug=%s)", errs.ErrScholarshipSlugTaken, params.Slug)
			case "23503":
				return sqlc.Scholarship{}, errs.ErrScholarshipProviderNotFound
			}
		}
		return sqlc.Scholarship{}, err
	}

	if len(ids.EducationLevelIDs) > 0 {
		if err = q.InsertScholarshipEducationLevels(ctx, sqlc.InsertScholarshipEducationLevelsParams{
			ScholarshipID: row.ID,
			Column2:       ids.EducationLevelIDs,
		}); err != nil {
			return sqlc.Scholarship{}, err
		}
	}
	if len(ids.MajorIDs) > 0 {
		if err = q.InsertScholarshipMajors(ctx, sqlc.InsertScholarshipMajorsParams{
			ScholarshipID: row.ID,
			Column2:       ids.MajorIDs,
		}); err != nil {
			return sqlc.Scholarship{}, err
		}
	}
	if len(ids.DemographicIDs) > 0 {
		if err = q.InsertScholarshipDemographics(ctx, sqlc.InsertScholarshipDemographicsParams{
			ScholarshipID: row.ID,
			Column2:       ids.DemographicIDs,
		}); err != nil {
			return sqlc.Scholarship{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return sqlc.Scholarship{}, err
	}
	return row, nil
}

func (r *scholarshipRepository) GetByID(ctx context.Context, id string) (sqlc.Scholarship, error) {
	row, err := r.queries.GetScholarshipByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Scholarship{}, errs.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return sqlc.Scholarship{}, errs.ErrNotFound
		}
		return sqlc.Scholarship{}, err
	}
	return row, nil
}

func (r *scholarshipRepository) Delete(ctx context.Context, id string) error {
	rows, err := r.queries.DeleteScholarship(ctx, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return errs.ErrNotFound
		}
		return err
	}
	if rows == 0 {
		return errs.ErrNotFound
	}
	return nil
}

func (r *scholarshipRepository) Search(ctx context.Context, q string) ([]sqlc.SearchScholarshipsRow, error) {
	return r.queries.SearchScholarships(ctx, sqlc.SearchScholarshipsParams{
		Query:       &q,
		ResultLimit: int32(maxSearchResults),
	})
}

func (r *scholarshipRepository) Publish(ctx context.Context, id string) (sqlc.Scholarship, error) {
	row, err := r.queries.PublishScholarship(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Scholarship{}, errs.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return sqlc.Scholarship{}, errs.ErrNotFound
		}
		return sqlc.Scholarship{}, err
	}
	return row, nil
}

func (r *scholarshipRepository) Get(ctx context.Context, q pagination.Query, f Filters) ([]ScholarshipListItem, int64, error) {
	where, args := buildScholarshipsWhere(f)

	var total int64
	countSQL := "SELECT COUNT(*) FROM scholarships s WHERE 1=1" + where
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count scholarships: %w", err)
	}

	listArgs := append(append([]any{}, args...), q.Limit(), q.Offset())
	listSQL := fmt.Sprintf(
		"SELECT s.id, s.title, s.slug, s.award_amount, COALESCE(s.logo, '') AS logo, COALESCE(s.provider_type, '') AS provider_type, COALESCE(s.provider_name, '') AS provider_name, COALESCE(s.country, '') AS country, s.application_deadline, COALESCE(s.min_gpa, 0)::float8 AS min_gpa, s.requires_financial_need, s.essay_required, s.is_renewable, s.is_popular, s.is_featured FROM scholarships s WHERE 1=1%s ORDER BY s.created_at DESC LIMIT $%d OFFSET $%d",
		where, len(listArgs)-1, len(listArgs),
	)

	rows, err := r.pool.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list scholarships: %w", err)
	}
	defer rows.Close()

	items, err := collectScholarships(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("scan scholarships: %w", err)
	}
	return items, total, nil
}

func collectScholarships(rows pgx.Rows) ([]ScholarshipListItem, error) {
	items := []ScholarshipListItem{}
	for rows.Next() {
		var s ScholarshipListItem
		if err := rows.Scan(
			&s.ID, &s.Title, &s.Slug, &s.AwardAmount,
			&s.Logo, &s.ProviderType, &s.ProviderName, &s.Country,
			&s.ApplicationDeadline, &s.MinGpa,
			&s.RequiresFinancialNeed, &s.EssayRequired, &s.IsRenewable,
			&s.IsPopular, &s.IsFeatured,
		); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

// buildScholarshipsWhere returns a parameterized WHERE fragment (with a leading
// " AND ") and the matching args. All clauses are AND-ed — a row must satisfy
// every supplied filter. Multi-value facets OR internally via ANY. Empty
// Filters produces empty output.
func buildScholarshipsWhere(f Filters) (string, []any) {
	var clauses []string
	var args []any

	add := func(clause string, val any) {
		args = append(args, val)
		clauses = append(clauses, fmt.Sprintf(clause, len(args)))
	}

	if f.Status != "" {
		add("s.status = $%d", f.Status)
	}
	if v := f.IsPopular; v != nil {
		add("s.is_popular = $%d", *v)
	}
	if v := f.IsFeatured; v != nil {
		add("s.is_featured = $%d", *v)
	}
	if f.ProviderType != "" {
		add("s.provider_type = $%d", f.ProviderType)
	}
	if f.State != "" {
		add("s.state = $%d", f.State)
	}
	if f.City != "" {
		add("s.city = $%d", f.City)
	}

	countryValues := f.Countries
	if f.Country != "" {
		countryValues = append(countryValues, f.Country)
	}
	if len(countryValues) > 0 {
		add("s.country = ANY($%d::text[])", countryValues)
	}

	// EXISTS subquery per multi-value facet — single index hit beats a join per
	// row. Filters by lookup name (filters.go pre-translated slugs to names).
	addExists := func(joinTable, lookupTable, idColumn string, values []string) {
		if len(values) == 0 {
			return
		}
		args = append(args, values)
		clauses = append(clauses, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM %s sl JOIN %s l ON l.id = sl.%s WHERE sl.scholarship_id = s.id AND l.name = ANY($%d::text[]))",
			joinTable, lookupTable, idColumn, len(args),
		))
	}
	addExists("scholarship_education_levels", "education_levels", "education_level_id", f.EducationLevels)
	addExists("scholarship_majors", "majors", "major_id", f.Majors)
	addExists("scholarship_demographics", "demographics", "demographic_id", f.Demographics)

	if f.MinGpa != nil {
		add("(s.min_gpa IS NULL OR s.min_gpa <= $%d)", *f.MinGpa)
	}
	if f.AwardMin != nil {
		add("s.award_min >= $%d", *f.AwardMin)
	}
	if f.AwardMax != nil {
		add("s.award_max <= $%d", *f.AwardMax)
	}
	if v := f.RequiresFinancialNeed; v != nil {
		add("s.requires_financial_need = $%d", *v)
	}
	if v := f.EssayRequired; v != nil {
		add("s.essay_required = $%d", *v)
	}
	if v := f.NoRecommendation; v != nil && *v {
		clauses = append(clauses, "s.recommendation_letters_required = 0")
	}
	if v := f.PortfolioRequired; v != nil {
		add("s.portfolio_required = $%d", *v)
	}
	if v := f.IsRenewable; v != nil {
		add("s.is_renewable = $%d", *v)
	}
	if f.DeadlineAfter != nil {
		add("s.application_deadline >= $%d", *f.DeadlineAfter)
	}
	if f.DeadlineBefore != nil {
		add("s.application_deadline <= $%d", *f.DeadlineBefore)
	}
	if v := f.OpenNow; v != nil && *v {
		clauses = append(clauses, "(s.application_open_date IS NULL OR s.application_open_date <= NOW()) AND (s.application_deadline IS NULL OR s.application_deadline >= NOW())")
	}

	if len(clauses) == 0 {
		return "", nil
	}
	return " AND " + strings.Join(clauses, " AND "), args
}

func (r *scholarshipRepository) Patch(ctx context.Context, id string, req *UpdateScholarshipRequest) (sqlc.Scholarship, error) {
	var err error
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return sqlc.Scholarship{}, err
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
		existing, gerr := getExisting(ctx, ids)
		if gerr != nil {
			return
		}
		if m := findMissing(existing, ids); len(m) > 0 {
			missing[table] = m
		}
	}
	if req.EducationLevelIDs != nil {
		collectMissing("education_levels", *req.EducationLevelIDs, q.GetExistingEducationLevelIDs)
	}
	if req.MajorIDs != nil {
		collectMissing("majors", *req.MajorIDs, q.GetExistingMajorIDs)
	}
	if req.DemographicIDs != nil {
		collectMissing("demographics", *req.DemographicIDs, q.GetExistingDemographicIDs)
	}
	if len(missing) > 0 {
		return sqlc.Scholarship{}, &errs.InvalidReferencesError{References: missing}
	}

	sets := []string{}
	args := []any{}
	addSet := func(col string, val any) {
		args = append(args, val)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}

	if req.Title != nil {
		addSet("title", *req.Title)
	}
	if req.Slug != nil {
		addSet("slug", *req.Slug)
	}
	if req.Description != nil {
		addSet("description", *req.Description)
	}
	if req.AwardAmount != nil {
		addSet("award_amount", *req.AwardAmount)
	}
	if req.AwardMin != nil {
		addSet("award_min", nullableNumeric(*req.AwardMin))
	}
	if req.AwardMax != nil {
		addSet("award_max", nullableNumeric(*req.AwardMax))
	}
	if req.Logo != nil {
		addSet("logo", *req.Logo)
	}
	if req.NumberOfAwards != nil {
		addSet("number_of_awards", *req.NumberOfAwards)
	}
	if req.IsRenewable != nil {
		addSet("is_renewable", *req.IsRenewable)
	}
	if req.MinGpa != nil {
		addSet("min_gpa", nullableNumeric(*req.MinGpa))
	}
	if req.RequiresFinancialNeed != nil {
		addSet("requires_financial_need", *req.RequiresFinancialNeed)
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
	if req.ApplicationOpenDate != nil {
		addSet("application_open_date", *req.ApplicationOpenDate)
	}
	if req.ApplicationDeadline != nil {
		addSet("application_deadline", *req.ApplicationDeadline)
	}
	if req.AwardNotificationDate != nil {
		addSet("award_notification_date", *req.AwardNotificationDate)
	}
	if req.EssayRequired != nil {
		addSet("essay_required", *req.EssayRequired)
	}
	if req.EssayPrompt != nil {
		addSet("essay_prompt", *req.EssayPrompt)
	}
	if req.RecommendationLettersRequired != nil {
		addSet("recommendation_letters_required", *req.RecommendationLettersRequired)
	}
	if req.TranscriptRequirement != nil {
		addSet("transcript_requirement", *req.TranscriptRequirement)
	}
	if req.PortfolioRequired != nil {
		addSet("portfolio_required", *req.PortfolioRequired)
	}
	if req.ApplicationUrl != nil {
		addSet("application_url", *req.ApplicationUrl)
	}
	if req.ProviderType != nil {
		addSet("provider_type", *req.ProviderType)
	}
	if req.ProviderName != nil {
		addSet("provider_name", *req.ProviderName)
	}
	if req.ContactEmail != nil {
		addSet("contact_email", *req.ContactEmail)
	}
	if req.ContactPhone != nil {
		addSet("contact_phone", *req.ContactPhone)
	}
	if req.InternalNotes != nil {
		addSet("internal_notes", *req.InternalNotes)
	}
	if req.UniversityID != nil {
		addSet("university_id", uuidFromString(*req.UniversityID))
	}
	if req.CollegeID != nil {
		addSet("college_id", uuidFromString(*req.CollegeID))
	}
	if req.SeoTitle != nil {
		addSet("seo_title", *req.SeoTitle)
	}
	if req.SeoDescription != nil {
		addSet("seo_description", *req.SeoDescription)
	}
	if req.IsPopular != nil {
		addSet("is_popular", *req.IsPopular)
	}
	if req.IsFeatured != nil {
		addSet("is_featured", *req.IsFeatured)
	}

	sets = append(sets, "updated_at = now()")
	args = append(args, id)
	updateSQL := fmt.Sprintf("UPDATE scholarships SET %s WHERE id = $%d", strings.Join(sets, ", "), len(args))

	ct, execErr := tx.Exec(ctx, updateSQL, args...)
	if execErr != nil {
		err = execErr
		var pgErr *pgconn.PgError
		if errors.As(execErr, &pgErr) {
			switch {
			case pgErr.Code == "23505" && req.Slug != nil:
				return sqlc.Scholarship{}, fmt.Errorf("%w (slug=%s)", errs.ErrScholarshipSlugTaken, *req.Slug)
			case pgErr.Code == "23503":
				return sqlc.Scholarship{}, errs.ErrScholarshipProviderNotFound
			case pgErr.Code == "22P02":
				return sqlc.Scholarship{}, errs.ErrNotFound
			}
		}
		return sqlc.Scholarship{}, execErr
	}
	if ct.RowsAffected() == 0 {
		err = errs.ErrNotFound
		return sqlc.Scholarship{}, err
	}

	replaceLookup := func(ids []string, deleteFn func(context.Context, string) error, insertFn func(context.Context, []string) error) error {
		if derr := deleteFn(ctx, id); derr != nil {
			return derr
		}
		if len(ids) > 0 {
			if ierr := insertFn(ctx, ids); ierr != nil {
				return ierr
			}
		}
		return nil
	}

	if req.EducationLevelIDs != nil {
		if err = replaceLookup(*req.EducationLevelIDs, q.DeleteScholarshipEducationLevels, func(ctx context.Context, ids []string) error {
			return q.InsertScholarshipEducationLevels(ctx, sqlc.InsertScholarshipEducationLevelsParams{ScholarshipID: id, Column2: ids})
		}); err != nil {
			return sqlc.Scholarship{}, err
		}
	}
	if req.MajorIDs != nil {
		if err = replaceLookup(*req.MajorIDs, q.DeleteScholarshipMajors, func(ctx context.Context, ids []string) error {
			return q.InsertScholarshipMajors(ctx, sqlc.InsertScholarshipMajorsParams{ScholarshipID: id, Column2: ids})
		}); err != nil {
			return sqlc.Scholarship{}, err
		}
	}
	if req.DemographicIDs != nil {
		if err = replaceLookup(*req.DemographicIDs, q.DeleteScholarshipDemographics, func(ctx context.Context, ids []string) error {
			return q.InsertScholarshipDemographics(ctx, sqlc.InsertScholarshipDemographicsParams{ScholarshipID: id, Column2: ids})
		}); err != nil {
			return sqlc.Scholarship{}, err
		}
	}

	// Re-fetch the updated row (junction changes don't affect the row itself) so
	// callers get the full, current record back.
	row, ferr := q.GetScholarshipByID(ctx, id)
	if ferr != nil {
		err = ferr
		return sqlc.Scholarship{}, ferr
	}

	if err = tx.Commit(ctx); err != nil {
		return sqlc.Scholarship{}, err
	}
	return row, nil
}

func (r *scholarshipRepository) GetScholarshipEducationLevels(ctx context.Context, scholarshipID string) ([]sqlc.EducationLevel, error) {
	return r.queries.GetScholarshipEducationLevels(ctx, scholarshipID)
}

func (r *scholarshipRepository) GetScholarshipMajors(ctx context.Context, scholarshipID string) ([]sqlc.Major, error) {
	return r.queries.GetScholarshipMajors(ctx, scholarshipID)
}

func (r *scholarshipRepository) GetScholarshipDemographics(ctx context.Context, scholarshipID string) ([]sqlc.Demographic, error) {
	return r.queries.GetScholarshipDemographics(ctx, scholarshipID)
}

func (r *scholarshipRepository) GetEducationLevels(ctx context.Context) ([]sqlc.EducationLevel, error) {
	return r.queries.GetEducationLevels(ctx)
}

func (r *scholarshipRepository) GetDemographics(ctx context.Context) ([]sqlc.Demographic, error) {
	return r.queries.GetDemographics(ctx)
}

func (r *scholarshipRepository) GetMajors(ctx context.Context) ([]sqlc.Major, error) {
	return r.queries.GetMajors(ctx)
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

	existing, err := q.GetExistingEducationLevelIDs(ctx, ids.EducationLevelIDs)
	if err != nil {
		return nil, err
	}
	record("education_levels", existing, ids.EducationLevelIDs)

	existing, err = q.GetExistingMajorIDs(ctx, ids.MajorIDs)
	if err != nil {
		return nil, err
	}
	record("majors", existing, ids.MajorIDs)

	existing, err = q.GetExistingDemographicIDs(ctx, ids.DemographicIDs)
	if err != nil {
		return nil, err
	}
	record("demographics", existing, ids.DemographicIDs)

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
