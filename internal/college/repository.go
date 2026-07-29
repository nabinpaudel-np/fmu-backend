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

type lookupIDs struct {
	DegreeLevelIDs []string
	MajorIDs       []string
	StudyFormatIDs []string
}

type CollegeRepository interface {
	Create(ctx context.Context, params sqlc.CreateCollegeParams, ids lookupIDs) (sqlc.College, error)
	Update(ctx context.Context, id string, req *UpdateCollegeRequest) (sqlc.College, error)
	List(ctx context.Context, q pagination.Query, f Filters) ([]sqlc.College, int64, error)
	GetByID(ctx context.Context, id string) (sqlc.College, error)
	ListByUniversity(ctx context.Context, universityID string, q pagination.Query) ([]sqlc.College, int64, error)
	Search(ctx context.Context, q string) ([]sqlc.SearchCollegesRow, error)
	RepresentedIDs(ctx context.Context, ids []string) (map[string]struct{}, error)
	Publish(ctx context.Context, id string) (sqlc.College, error)
	GetCollegeDegreeLevels(ctx context.Context, collegeID string) ([]sqlc.DegreeLevel, error)
	GetCollegeMajors(ctx context.Context, collegeID string) ([]sqlc.Major, error)
	GetCollegeStudyFormats(ctx context.Context, collegeID string) ([]sqlc.StudyFormat, error)
}

type collegeRepository struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
}

const maxCollegeSearchResults = 50

// Keep this list explicit because ALTER TABLE ADD COLUMN can make runtime order
// differ from schema.sql's declaration order.
const collegeColumnList = "id, name, slug, university_id, overview, excerpt, country, state, city, full_location, cover_image, logo, institution_type, campus_setting, contact_email, contact_phone, website, zipcode, founded_year, campus_size, gallery_images, is_popular, is_featured, full_address, maps_url, seo_title, seo_description, status, published_at, created_at, updated_at"

func NewCollegeRepository(queries *sqlc.Queries, pool *pgxpool.Pool) CollegeRepository {
	return &collegeRepository{
		queries: queries,
		pool:    pool,
	}
}

func (r *collegeRepository) Create(ctx context.Context, params sqlc.CreateCollegeParams, ids lookupIDs) (sqlc.College, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return sqlc.College{}, err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	q := r.queries.WithTx(tx)

	if missing, mErr := validateCollegeReferences(ctx, q, ids); mErr != nil {
		return sqlc.College{}, mErr
	} else if len(missing) > 0 {
		return sqlc.College{}, &errs.InvalidReferencesError{References: missing}
	}

	row, err := q.CreateCollege(ctx, params)
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

	if len(ids.DegreeLevelIDs) > 0 {
		if err = q.InsertCollegeDegreeLevels(ctx, sqlc.InsertCollegeDegreeLevelsParams{
			CollegeID:    row.ID,
			Column2:      ids.DegreeLevelIDs,
		}); err != nil {
			return sqlc.College{}, err
		}
	}
	if len(ids.MajorIDs) > 0 {
		if err = q.InsertCollegeMajors(ctx, sqlc.InsertCollegeMajorsParams{
			CollegeID: row.ID,
			Column2:   ids.MajorIDs,
		}); err != nil {
			return sqlc.College{}, err
		}
	}
	if len(ids.StudyFormatIDs) > 0 {
		if err = q.InsertCollegeStudyFormats(ctx, sqlc.InsertCollegeStudyFormatsParams{
			CollegeID: row.ID,
			Column2:   ids.StudyFormatIDs,
		}); err != nil {
			return sqlc.College{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
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
			&c.ID, &c.Name, &c.Slug, &c.UniversityID, &c.Overview, &c.Excerpt,
			&c.Country, &c.State, &c.City, &c.FullLocation,
			&c.CoverImage, &c.Logo, &c.InstitutionType, &c.CampusSetting,
			&c.ContactEmail, &c.ContactPhone, &c.Website, &c.Zipcode,
			&c.FoundedYear, &c.CampusSize, &c.GalleryImages,
			&c.IsPopular, &c.IsFeatured,
			&c.FullAddress, &c.MapsUrl, &c.SeoTitle, &c.SeoDescription,
			&c.Status, &c.PublishedAt,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

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

	// Status scopes the whole query (AND-ed in), the rest are AND-ed too.
	if f.Status != "" {
		args = append(args, f.Status)
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	if v := f.IsPopular; v != nil {
		args = append(args, *v)
		clauses = append(clauses, fmt.Sprintf("is_popular = $%d", len(args)))
	}
	if v := f.IsFeatured; v != nil {
		args = append(args, *v)
		clauses = append(clauses, fmt.Sprintf("is_featured = $%d", len(args)))
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

func (r *collegeRepository) Update(ctx context.Context, id string, req *UpdateCollegeRequest) (sqlc.College, error) {
	var err error
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return sqlc.College{}, err
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
		existing, gErr := getExisting(ctx, ids)
		if gErr != nil {
			err = gErr
			return
		}
		if m := findMissing(existing, ids); len(m) > 0 {
			missing[table] = m
		}
	}

	if req.DegreeLevelIDs != nil {
		collectMissing("degree_levels", *req.DegreeLevelIDs, q.GetExistingCollegeDegreeLevelIDs)
	}
	if req.MajorIDs != nil {
		collectMissing("majors", *req.MajorIDs, q.GetExistingCollegeMajorIDs)
	}
	if req.StudyFormatIDs != nil {
		collectMissing("study_formats", *req.StudyFormatIDs, q.GetExistingCollegeStudyFormatIDs)
	}
	if err != nil {
		return sqlc.College{}, err
	}
	if len(missing) > 0 {
		return sqlc.College{}, &errs.InvalidReferencesError{References: missing}
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
	if req.FullAddress != nil {
		addSet("full_address", *req.FullAddress)
	}
	if req.CoverImage != nil {
		addSet("cover_image", *req.CoverImage)
	}
	if req.Logo != nil {
		addSet("logo", *req.Logo)
	}
	if req.MapsUrl != nil {
		addSet("maps_url", *req.MapsUrl)
	}
	if req.InstitutionType != nil {
		addSet("institution_type", *req.InstitutionType)
	}
	if req.CampusSetting != nil {
		addSet("campus_setting", *req.CampusSetting)
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
	if req.SeoTitle != nil {
		addSet("seo_title", *req.SeoTitle)
	}
	if req.SeoDescription != nil {
		addSet("seo_description", *req.SeoDescription)
	}

	var row sqlc.College
	if len(sets) == 0 {
		// Nothing scalar to update. Either the request is only changing
		// lookup associations, or it's empty. Fetch the current row so
		// callers always get a College back.
		if getErr := tx.QueryRow(ctx, "SELECT "+collegeColumnList+" FROM colleges WHERE id = $1", id).Scan(
			&row.ID, &row.Name, &row.Slug, &row.UniversityID, &row.Overview, &row.Excerpt,
			&row.Country, &row.State, &row.City, &row.FullLocation,
			&row.CoverImage, &row.Logo, &row.InstitutionType, &row.CampusSetting,
			&row.ContactEmail, &row.ContactPhone, &row.Website, &row.Zipcode,
			&row.FoundedYear, &row.CampusSize, &row.GalleryImages,
			&row.IsPopular, &row.IsFeatured,
			&row.FullAddress, &row.MapsUrl, &row.SeoTitle, &row.SeoDescription,
			&row.Status, &row.PublishedAt,
			&row.CreatedAt, &row.UpdatedAt,
		); getErr != nil {
			if errors.Is(getErr, pgx.ErrNoRows) {
				return sqlc.College{}, errs.ErrNotFound
			}
			return sqlc.College{}, getErr
		}
	} else {
		sets = append(sets, "updated_at = now()")
		args = append(args, id)
		sql := fmt.Sprintf(
			`UPDATE colleges SET %s WHERE id = $%d
			 RETURNING %s`,
			strings.Join(sets, ", "), len(args), collegeColumnList,
		)

		scanErr := tx.QueryRow(ctx, sql, args...).Scan(
			&row.ID, &row.Name, &row.Slug, &row.UniversityID, &row.Overview, &row.Excerpt,
			&row.Country, &row.State, &row.City, &row.FullLocation,
			&row.CoverImage, &row.Logo, &row.InstitutionType, &row.CampusSetting,
			&row.ContactEmail, &row.ContactPhone, &row.Website, &row.Zipcode,
			&row.FoundedYear, &row.CampusSize, &row.GalleryImages,
			&row.IsPopular, &row.IsFeatured,
			&row.FullAddress, &row.MapsUrl, &row.SeoTitle, &row.SeoDescription,
			&row.Status, &row.PublishedAt,
			&row.CreatedAt, &row.UpdatedAt,
		)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return sqlc.College{}, errs.ErrNotFound
			}
			var pgErr *pgconn.PgError
			if errors.As(scanErr, &pgErr) && pgErr.Code == "23505" {
				return sqlc.College{}, fmt.Errorf("%w (slug=%v)", errs.ErrCollegeSlugTaken, deref(req.Slug))
			}
			if errors.As(scanErr, &pgErr) && pgErr.Code == "22P02" {
				return sqlc.College{}, errs.ErrNotFound
			}
			return sqlc.College{}, scanErr
		}
	}

	replaceLookup := func(ids []string, deleteFn func(context.Context, string) error, insertFn func(context.Context, []string) error) error {
		if dErr := deleteFn(ctx, id); dErr != nil {
			return dErr
		}
		if len(ids) > 0 {
			if iErr := insertFn(ctx, ids); iErr != nil {
				return iErr
			}
		}
		return nil
	}

	if req.DegreeLevelIDs != nil {
		if err = replaceLookup(*req.DegreeLevelIDs, q.DeleteCollegeDegreeLevels, func(ctx context.Context, ids []string) error {
			return q.InsertCollegeDegreeLevels(ctx, sqlc.InsertCollegeDegreeLevelsParams{
				CollegeID: row.ID,
				Column2:   ids,
			})
		}); err != nil {
			return sqlc.College{}, err
		}
	}
	if req.MajorIDs != nil {
		if err = replaceLookup(*req.MajorIDs, q.DeleteCollegeMajors, func(ctx context.Context, ids []string) error {
			return q.InsertCollegeMajors(ctx, sqlc.InsertCollegeMajorsParams{
				CollegeID: row.ID,
				Column2:   ids,
			})
		}); err != nil {
			return sqlc.College{}, err
		}
	}
	if req.StudyFormatIDs != nil {
		if err = replaceLookup(*req.StudyFormatIDs, q.DeleteCollegeStudyFormats, func(ctx context.Context, ids []string) error {
			return q.InsertCollegeStudyFormats(ctx, sqlc.InsertCollegeStudyFormatsParams{
				CollegeID: row.ID,
				Column2:   ids,
			})
		}); err != nil {
			return sqlc.College{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return sqlc.College{}, err
	}

	return row, nil
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

func (r *collegeRepository) RepresentedIDs(ctx context.Context, ids []string) (map[string]struct{}, error) {
	if len(ids) == 0 {
		return map[string]struct{}{}, nil
	}
	rows, err := r.queries.ListRepresentedCollegeIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list represented college ids: %w", err)
	}
	set := make(map[string]struct{}, len(rows))
	for _, id := range rows {
		set[id] = struct{}{}
	}
	return set, nil
}

func (r *collegeRepository) Publish(ctx context.Context, id string) (sqlc.College, error) {
	row, err := r.queries.PublishCollege(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.College{}, errs.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return sqlc.College{}, errs.ErrNotFound
		}
		return sqlc.College{}, err
	}
	return row, nil
}

func (r *collegeRepository) GetCollegeDegreeLevels(ctx context.Context, collegeID string) ([]sqlc.DegreeLevel, error) {
	return r.queries.GetCollegeDegreeLevels(ctx, collegeID)
}

func (r *collegeRepository) GetCollegeMajors(ctx context.Context, collegeID string) ([]sqlc.Major, error) {
	return r.queries.GetCollegeMajors(ctx, collegeID)
}

func (r *collegeRepository) GetCollegeStudyFormats(ctx context.Context, collegeID string) ([]sqlc.StudyFormat, error) {
	return r.queries.GetCollegeStudyFormats(ctx, collegeID)
}

// validateCollegeReferences returns a map of resource name → missing IDs
// for each lookup table referenced by ids. An empty map (and nil error)
// means every requested ID exists. Mirrors university.validateReferences but
// scoped to the three lookup tables colleges use.
func validateCollegeReferences(ctx context.Context, q *sqlc.Queries, ids lookupIDs) (map[string][]string, error) {
	var missing map[string][]string

	record := func(table string, existing, requested []string) {
		if m := findMissing(existing, requested); len(m) > 0 {
			if missing == nil {
				missing = make(map[string][]string)
			}
			missing[table] = m
		}
	}

	existing, err := q.GetExistingCollegeDegreeLevelIDs(ctx, ids.DegreeLevelIDs)
	if err != nil {
		return nil, err
	}
	record("degree_levels", existing, ids.DegreeLevelIDs)

	existing, err = q.GetExistingCollegeMajorIDs(ctx, ids.MajorIDs)
	if err != nil {
		return nil, err
	}
	record("majors", existing, ids.MajorIDs)

	existing, err = q.GetExistingCollegeStudyFormatIDs(ctx, ids.StudyFormatIDs)
	if err != nil {
		return nil, err
	}
	record("study_formats", existing, ids.StudyFormatIDs)

	return missing, nil
}

// findMissing returns the subset of `requested` that don't appear in `existing`.
// `existing` is expected to be the dedup'd set of IDs that actually exist in
// the lookup table — the caller provides that set so we don't have to scan
// each ID one at a time.
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
