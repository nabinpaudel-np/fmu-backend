package college

import (
	"context"
	"errors"
	"fmt"
	"strconv"
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
	BulkInsertColleges(ctx context.Context, plans []collegeBulkInsertPlan) ([]InsertedCollege, error)
	ListExistingCollegeSlugs(ctx context.Context, slugs []string) ([]string, error)
	GetDegreeLevels(ctx context.Context) ([]sqlc.DegreeLevel, error)
	GetMajors(ctx context.Context) ([]sqlc.Major, error)
	GetStudyFormats(ctx context.Context) ([]sqlc.StudyFormat, error)
}

// InsertedCollege is the minimal projection returned by the bulk insert
// path. The service uses it to reconcile ON CONFLICT skips against the
// input plan list.
type InsertedCollege struct {
	ID   string
	Slug string
}

type collegeRepository struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
}

const maxCollegeSearchResults = 50

// Keep this list explicit because ALTER TABLE ADD COLUMN can make runtime order
// differ from schema.sql's declaration order.
const collegeColumnList = "id, name, slug, university_id, overview, excerpt, country, continent, state, city, full_location, cover_image, logo, institution_type, campus_setting, contact_email, contact_phone, website, zipcode, founded_year, campus_size, gallery_images, is_popular, is_featured, full_address, maps_url, seo_title, seo_description, status, published_at, created_at, updated_at"

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
			return sqlc.College{}, fmt.Errorf("%w (university_id=%s)", errs.ErrCollegeUniversityNotFound, fromPgUUID(params.UniversityID))
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
			&c.Country, &c.Continent, &c.State, &c.City, &c.FullLocation,
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
	if req.UniversityID != nil {
		if exists, uErr := universityExists(ctx, tx, *req.UniversityID); uErr != nil {
			err = uErr
			return sqlc.College{}, err
		} else if !exists {
			missing["universities"] = []string{*req.UniversityID}
		}
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
	if req.Continent != nil {
		addSet("continent", *req.Continent)
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
	if req.UniversityID != nil {
		addSet("university_id", toPgUUID(*req.UniversityID))
	}

	var row sqlc.College
	if len(sets) == 0 {
		// Nothing scalar to update. Either the request is only changing
		// lookup associations, or it's empty. Fetch the current row so
		// callers always get a College back.
		if getErr := tx.QueryRow(ctx, "SELECT "+collegeColumnList+" FROM colleges WHERE id = $1", id).Scan(
			&row.ID, &row.Name, &row.Slug, &row.UniversityID, &row.Overview, &row.Excerpt,
			&row.Country, &row.Continent, &row.State, &row.City, &row.FullLocation,
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
			&row.Country, &row.Continent, &row.State, &row.City, &row.FullLocation,
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
	uid := toPgUUID(universityID)
	total, err := r.queries.CountCollegesByUniversity(ctx, uid)
	if err != nil {
		return nil, 0, fmt.Errorf("count colleges by university: %w", err)
	}
	rows, err := r.queries.ListCollegesByUniversity(ctx, sqlc.ListCollegesByUniversityParams{
		UniversityID: uid,
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

func (r *collegeRepository) GetDegreeLevels(ctx context.Context) ([]sqlc.DegreeLevel, error) {
	return r.queries.GetDegreeLevels(ctx)
}

func (r *collegeRepository) GetMajors(ctx context.Context) ([]sqlc.Major, error) {
	return r.queries.GetMajors(ctx)
}

func (r *collegeRepository) GetStudyFormats(ctx context.Context) ([]sqlc.StudyFormat, error) {
	return r.queries.GetStudyFormats(ctx)
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

// universityExists reports whether a row with the supplied UUID exists in
// the universities table. Used by Update to validate an orphan college's
// new parent before flipping the FK — a malformed UUID or unknown parent
// surfaces as an InvalidReferencesError instead of a 23503 FK violation
// at COMMIT time, so the admin gets a clear error message.
func universityExists(ctx context.Context, tx pgx.Tx, id string) (bool, error) {
	var found bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM universities WHERE id = $1)`, id).Scan(&found); err != nil {
		return false, err
	}
	return found, nil
}

// ListExistingCollegeSlugs returns the subset of slugs that already exist
// in the colleges table. Used by the bulk-upload service to skip
// already-imported rows so re-uploading the same CSV is a no-op rather
// than a duplicate-key error.
func (r *collegeRepository) ListExistingCollegeSlugs(ctx context.Context, slugs []string) ([]string, error) {
	if len(slugs) == 0 {
		return nil, nil
	}
	rows, err := r.queries.ListExistingCollegeSlugs(ctx, slugs)
	if err != nil {
		return nil, fmt.Errorf("list existing college slugs: %w", err)
	}
	return rows, nil
}

// collegeBulkInsertPlan is the per-row payload that BulkInsertColleges
// needs: the column values + the junction IDs to wire up after the row
// lands. Keeping lookupIDs on the plan avoids re-walking it at
// junction-insert time.
type collegeBulkInsertPlan struct {
	params    sqlc.CreateCollegeParams
	lookupIDs lookupIDs
}

// bulkInsertChunkSize bounds the number of VALUES tuples per round-trip.
// 500 matches the university bulk insert; smaller chunks add overhead,
// larger ones hit pgx's parameter-array limits.
const bulkInsertChunkSize = 500

// BulkInsertColleges inserts every supplied college in a single
// transaction using a chunked multi-row INSERT … ON CONFLICT DO NOTHING.
// ON CONFLICT swallows the race where another admin inserts the same
// slug between our pre-flight ListExistingCollegeSlugs check and this
// call — the row is simply skipped, matching "create colleges that
// don't exist".
//
// Junction rows are inserted per-college using the existing per-row
// helpers, keyed off the UUIDs Postgres assigns via gen_random_uuid() +
// RETURNING. This is the same pattern the single-row Create path uses.
func (r *collegeRepository) BulkInsertColleges(
	ctx context.Context,
	rows []collegeBulkInsertPlan,
) ([]InsertedCollege, error) {
	if len(rows) == 0 {
		return nil, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	q := r.queries.WithTx(tx)

	inserted := make([]InsertedCollege, 0, len(rows))
	for start := 0; start < len(rows); start += bulkInsertChunkSize {
		end := start + bulkInsertChunkSize
		if end > len(rows) {
			end = len(rows)
		}
		chunk := rows[start:end]
		ids, chunkErr := r.bulkInsertCollegesChunk(ctx, tx, chunk)
		if chunkErr != nil {
			err = chunkErr
			return nil, err
		}
		inserted = append(inserted, ids...)

		for i, ins := range ids {
			plan := chunk[i]
			if len(plan.lookupIDs.DegreeLevelIDs) > 0 {
				if err = q.InsertCollegeDegreeLevels(ctx, sqlc.InsertCollegeDegreeLevelsParams{
					CollegeID: ins.ID,
					Column2:   plan.lookupIDs.DegreeLevelIDs,
				}); err != nil {
					return nil, err
				}
			}
			if len(plan.lookupIDs.MajorIDs) > 0 {
				if err = q.InsertCollegeMajors(ctx, sqlc.InsertCollegeMajorsParams{
					CollegeID: ins.ID,
					Column2:   plan.lookupIDs.MajorIDs,
				}); err != nil {
					return nil, err
				}
			}
			if len(plan.lookupIDs.StudyFormatIDs) > 0 {
				if err = q.InsertCollegeStudyFormats(ctx, sqlc.InsertCollegeStudyFormatsParams{
					CollegeID: ins.ID,
					Column2:   plan.lookupIDs.StudyFormatIDs,
				}); err != nil {
					return nil, err
				}
			}
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return inserted, nil
}

// bulkInsertCollegesChunk runs one multi-row INSERT for the supplied
// plans and returns the (id, slug) of every row that actually landed.
// Rows skipped by ON CONFLICT DO NOTHING don't appear in RETURNING and
// are absent from the result; the caller accounts for them via
// len(plan) − len(result).
//
// The column order mirrors CreateCollege's INSERT statement. Don't
// reorder without updating sqlc/queries/colleges.sql in lockstep.
func (r *collegeRepository) bulkInsertCollegesChunk(
	ctx context.Context,
	tx pgx.Tx,
	plans []collegeBulkInsertPlan,
) ([]InsertedCollege, error) {
	if len(plans) == 0 {
		return nil, nil
	}

	var b strings.Builder
	b.WriteString(`INSERT INTO colleges (
		name, slug, university_id, overview, excerpt,
		country, continent, state, city, full_location,
		cover_image, logo, institution_type, campus_setting,
		contact_email, contact_phone, website, zipcode,
		founded_year, campus_size, gallery_images,
		is_popular, is_featured,
		full_address, maps_url, seo_title, seo_description,
		status
	) VALUES `)

	const colsPerRow = 28
	args := make([]any, 0, len(plans)*colsPerRow)
	for i, plan := range plans {
		if i > 0 {
			b.WriteString(", ")
		}
		args = append(args,
			plan.params.Name,
			plan.params.Slug,
			plan.params.UniversityID,
			plan.params.Overview,
			plan.params.Excerpt,
			plan.params.Country,
			plan.params.Continent,
			plan.params.State,
			plan.params.City,
			plan.params.FullLocation,
			plan.params.CoverImage,
			plan.params.Logo,
			plan.params.InstitutionType,
			plan.params.CampusSetting,
			plan.params.ContactEmail,
			plan.params.ContactPhone,
			plan.params.Website,
			plan.params.Zipcode,
			plan.params.FoundedYear,
			plan.params.CampusSize,
			plan.params.GalleryImages,
			plan.params.IsPopular,
			plan.params.IsFeatured,
			plan.params.FullAddress,
			plan.params.MapsUrl,
			plan.params.SeoTitle,
			plan.params.SeoDescription,
			plan.params.Status,
		)
		b.WriteByte('(')
		for j := 0; j < colsPerRow; j++ {
			if j > 0 {
				b.WriteString(", ")
			}
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(j + 1 + i*colsPerRow))
		}
		b.WriteByte(')')
	}
	b.WriteString(` ON CONFLICT (slug) DO NOTHING RETURNING id, slug`)

	rows, err := tx.Query(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("bulk insert colleges: %w", err)
	}
	defer rows.Close()

	out := make([]InsertedCollege, 0, len(plans))
	for rows.Next() {
		var ins InsertedCollege
		if err := rows.Scan(&ins.ID, &ins.Slug); err != nil {
			return nil, err
		}
		out = append(out, ins)
	}
	return out, rows.Err()
}
