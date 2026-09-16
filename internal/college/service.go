package college

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
)

type CollegeService interface {
	Create(ctx context.Context, req *CreateCollegeRequest) (*CreateCollegeResponse, error)
	Update(ctx context.Context, id string, req *UpdateCollegeRequest) (*CreateCollegeResponse, error)
	List(ctx context.Context, q pagination.Query, f Filters) ([]CollegeListItem, int64, error)
	GetByID(ctx context.Context, id string) (*CollegeDetailResponse, error)
	ListByUniversity(ctx context.Context, universityID string, q pagination.Query) ([]CollegeListItem, int64, error)
	Search(ctx context.Context, q string) ([]CollegeSearchResult, error)
	RepresentedIDs(ctx context.Context, ids []string) (map[string]struct{}, error)
	Publish(ctx context.Context, id string) (*CreateCollegeResponse, error)
	BulkUpload(ctx context.Context, status string, rows []ParsedCollegeRow) (*CollegeBulkUploadResponse, error)
}

type collegeService struct {
	repo CollegeRepository
}

func NewCollegeService(repo CollegeRepository) CollegeService {
	return &collegeService{repo: repo}
}

func (s *collegeService) Create(ctx context.Context, req *CreateCollegeRequest) (*CreateCollegeResponse, error) {
	// Representatives can create colleges only under their own university.
	// Admins are unrestricted and may leave university_id empty to create
	// an orphan college (assignable later via PATCH). AuthMiddleware
	// injects claims into ctx so the body-level scope check lives next to
	// the data write.
	if claims, err := auth.ClaimsFromContext(ctx); err == nil && claims.Role == auth.RoleRepresentative {
		if req.UniversityID == "" {
			return nil, errs.ErrRepUniversityIDRequired
		}
		if claims.RepresentativeUniversityID == "" || claims.RepresentativeUniversityID != req.UniversityID {
			return nil, errs.ErrRepOutOfScope
		}
	}

	// Drafts only require name + slug. The CreateCollegeRequest struct has
	// `validate:"required"` on overview; university_id was relaxed to
	// `omitempty` so admins can stage orphan drafts. The publish endpoint
	// re-validates the full required-field set before flipping status to
	// "published" (which still requires a parent university).
	if req.Status == "draft" {
		if err := validateDraftCollege(req); err != nil {
			return nil, err
		}
	}

	row, err := s.repo.Create(ctx, toCreateCollegeParams(req), lookupIDs{
		DegreeLevelIDs: req.DegreeLevelIDs,
		MajorIDs:       req.MajorIDs,
		StudyFormatIDs: req.StudyFormatIDs,
	})
	if err != nil {
		log.Default().Printf("failed to create college: %v", err)
		return nil, err
	}
	return toCreateCollegeResponse(row), nil
}

func validateDraftCollege(req *CreateCollegeRequest) error {
	var missing []string
	if req.Name == "" {
		missing = append(missing, "name")
	}
	if req.Slug == "" {
		missing = append(missing, "slug")
	}
	if len(missing) > 0 {
		return &errs.PublishValidationError{Fields: missing}
	}
	return nil
}

func (s *collegeService) Publish(ctx context.Context, id string) (*CreateCollegeResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return nil, errs.ErrNotFound
		}
		log.Default().Printf("get college %s for publish: %v", id, err)
		return nil, err
	}
	if missing := requiredFieldsForCollegePublish(row); len(missing) > 0 {
		return nil, &errs.PublishValidationError{Fields: missing}
	}
	published, err := s.repo.Publish(ctx, id)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		log.Default().Printf("publish college %s: %v", id, err)
		return nil, err
	}
	return toCreateCollegeResponse(published), nil
}

// requiredFieldsForCollegePublish mirrors the `validate:"required"` tags on
// CreateCollegeRequest. Publish requires the row to be presentable to the
// public, so university_id + overview (the two DTO-required fields besides
// name/slug) must be filled.
func requiredFieldsForCollegePublish(c sqlc.College) []string {
	var missing []string
	if c.Name == "" {
		missing = append(missing, "name")
	}
	if c.Slug == "" {
		missing = append(missing, "slug")
	}
	if !c.UniversityID.Valid {
		missing = append(missing, "university_id")
	}
	if c.Overview == "" {
		missing = append(missing, "overview")
	}
	return missing
}

func (s *collegeService) List(ctx context.Context, q pagination.Query, f Filters) ([]CollegeListItem, int64, error) {
	rows, total, err := s.repo.List(ctx, q, f)
	if err != nil {
		log.Default().Printf("failed to list colleges: %v", err)
		return nil, 0, err
	}
	items := make([]CollegeListItem, len(rows))
	for i, row := range rows {
		items[i] = toCollegeListItem(row)
	}
	return items, total, nil
}

func (s *collegeService) GetByID(ctx context.Context, id string) (*CollegeDetailResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return nil, errs.ErrNotFound
		}
		log.Default().Printf("failed to get college %s: %v", id, err)
		return nil, err
	}

	degreeLevels, err := s.repo.GetCollegeDegreeLevels(ctx, id)
	if err != nil {
		return nil, err
	}
	majors, err := s.repo.GetCollegeMajors(ctx, id)
	if err != nil {
		return nil, err
	}
	studyFormats, err := s.repo.GetCollegeStudyFormats(ctx, id)
	if err != nil {
		return nil, err
	}

	return toCollegeDetailResponse(row, degreeLevels, majors, studyFormats), nil
}

func (s *collegeService) Update(ctx context.Context, id string, req *UpdateCollegeRequest) (*CreateCollegeResponse, error) {
	if claims, err := auth.ClaimsFromContext(ctx); err == nil && claims.Role == auth.RoleRepresentative {
		if req.Name != nil || req.Slug != nil {
			return nil, errs.ErrRepCannotChangeNameOrSlug
		}
	}

	row, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		if errors.Is(err, errs.ErrCollegeSlugTaken) {
			return nil, err
		}
		log.Default().Printf("failed to update college %s: %v", id, err)
		return nil, err
	}
	return toCreateCollegeResponse(row), nil
}

func (s *collegeService) ListByUniversity(ctx context.Context, universityID string, q pagination.Query) ([]CollegeListItem, int64, error) {
	rows, total, err := s.repo.ListByUniversity(ctx, universityID, q)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return nil, 0, errs.ErrNotFound
		}
		log.Default().Printf("failed to list colleges for university %s: %v", universityID, err)
		return nil, 0, err
	}
	items := make([]CollegeListItem, len(rows))
	for i, row := range rows {
		items[i] = toCollegeListItem(row)
	}
	return items, total, nil
}

func (s *collegeService) Search(ctx context.Context, q string) ([]CollegeSearchResult, error) {
	rows, err := s.repo.Search(ctx, q)
	if err != nil {
		log.Default().Printf("search colleges q=%q: %v", q, err)
		return nil, err
	}
	items := make([]CollegeSearchResult, len(rows))
	for i, row := range rows {
		items[i] = toCollegeSearchResult(row)
	}
	return items, nil
}

func (s *collegeService) RepresentedIDs(ctx context.Context, ids []string) (map[string]struct{}, error) {
	set, err := s.repo.RepresentedIDs(ctx, ids)
	if err != nil {
		log.Default().Printf("failed to list represented college ids: %v", err)
		return nil, err
	}
	return set, nil
}

// BulkUpload ingests a stream of already-parsed CSV rows, resolves name
// lookups, drops rows whose slug is already in the DB, and inserts the
// remainder inside a single transaction. Failure semantics mirror
// universityService.BulkUpload (all-or-nothing on validation errors,
// idempotent on slug). Bulk-uploaded colleges are orphans by design —
// their university_id is always NULL on insert; admins attach a parent
// later via PATCH /colleges/{id}.
func (s *collegeService) BulkUpload(ctx context.Context, status string, rows []ParsedCollegeRow) (*CollegeBulkUploadResponse, error) {
	if len(rows) == 0 {
		return &CollegeBulkUploadResponse{
			Created:         0,
			SkippedExisting: 0,
			Errors:          []CollegeRowError{},
		}, nil
	}

	validDegreeLevels, err := s.nameLookup(ctx, func(ctx context.Context) ([]namedLookup, error) {
		rows, err := s.repo.GetDegreeLevels(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]namedLookup, len(rows))
		for i, r := range rows {
			out[i] = namedLookup{ID: r.ID, Name: r.Name}
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	validMajors, err := s.nameLookup(ctx, func(ctx context.Context) ([]namedLookup, error) {
		rows, err := s.repo.GetMajors(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]namedLookup, len(rows))
		for i, r := range rows {
			out[i] = namedLookup{ID: r.ID, Name: r.Name}
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	validStudyFormats, err := s.nameLookup(ctx, func(ctx context.Context) ([]namedLookup, error) {
		rows, err := s.repo.GetStudyFormats(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]namedLookup, len(rows))
		for i, r := range rows {
			out[i] = namedLookup{ID: r.ID, Name: r.Name}
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}

	if nameErrs := ResolveNames(rows, validDegreeLevels, validMajors, validStudyFormats); len(nameErrs) > 0 {
		return nil, &errs.BulkValidationError{Errors: CapErrorCount(nameErrs)}
	}

	slugs := make([]string, len(rows))
	for i, row := range rows {
		slugs[i] = row.Slug
	}
	existing, err := s.repo.ListExistingCollegeSlugs(ctx, slugs)
	if err != nil {
		return nil, err
	}
	existingSet := make(map[string]struct{}, len(existing))
	for _, slug := range existing {
		existingSet[slug] = struct{}{}
	}

	var (
		toInsert        []ParsedCollegeRow
		skippedExisting int
	)
	for _, row := range rows {
		if _, dup := existingSet[row.Slug]; dup {
			skippedExisting++
			continue
		}
		toInsert = append(toInsert, row)
	}

	plans := make([]collegeBulkInsertPlan, 0, len(toInsert))
	for _, row := range toInsert {
		plans = append(plans, collegeBulkInsertPlan{
			params:    toCreateCollegeParams(parsedRowToRequest(row, status)),
			lookupIDs: lookupIDsFromParsedRow(row),
		})
	}

	inserted, err := s.repo.BulkInsertColleges(ctx, plans)
	if err != nil {
		return nil, err
	}

	actuallyInserted := len(inserted)
	raceSkipped := len(plans) - actuallyInserted

	return &CollegeBulkUploadResponse{
		Created:         actuallyInserted,
		SkippedExisting: skippedExisting + raceSkipped,
		Errors:          []CollegeRowError{},
	}, nil
}

// nameLookup fetches a lookup table and returns a lowercase-name → id map
// for ResolveNames. Centralized so the three calls share one place to
// handle errors / case-fold.
func (s *collegeService) nameLookup(
	ctx context.Context,
	fetch func(ctx context.Context) ([]namedLookup, error),
) (map[string]string, error) {
	rows, err := fetch(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[strings.ToLower(r.Name)] = r.ID
	}
	return out, nil
}

type namedLookup struct {
	ID   string
	Name string
}

// parsedRowToRequest converts a ParsedCollegeRow into the
// CreateCollegeRequest the repository expects. Lives here so the bulk
// parser doesn't need to import the college DTOs. UniversityID is left
// empty — bulk-uploaded colleges are always orphans.
func parsedRowToRequest(row ParsedCollegeRow, status string) *CreateCollegeRequest {
	if status == "" {
		status = StatusDraft
	}
	return &CreateCollegeRequest{
		Name:            row.Name,
		Slug:            row.Slug,
		Overview:        row.Overview,
		Excerpt:         row.Excerpt,
		Country:         row.Country,
		Continent:       row.Continent,
		State:           row.State,
		City:            row.City,
		FullLocation:    row.FullLocation,
		Zipcode:         row.Zipcode,
		FullAddress:     row.FullAddress,
		CoverImage:      row.CoverImage,
		Logo:            row.Logo,
		MapsUrl:         row.MapsUrl,
		GalleryImages:   row.GalleryImages,
		InstitutionType: row.InstitutionType,
		CampusSetting:   row.CampusSetting,
		ContactEmail:    row.ContactEmail,
		ContactPhone:    row.ContactPhone,
		Website:         row.Website,
		FoundedYear:     row.FoundedYear,
		CampusSize:      row.CampusSize,
		IsPopular:       row.IsPopular,
		IsFeatured:      row.IsFeatured,
		SeoTitle:        row.SeoTitle,
		SeoDescription:  row.SeoDescription,
		Status:          status,
		DegreeLevelIDs:  row.DegreeLevelIDs,
		MajorIDs:        row.MajorIDs,
		StudyFormatIDs:  row.StudyFormatIDs,
	}
}

// lookupIDsFromParsedRow mirrors parsedRowToRequest for the junction IDs.
func lookupIDsFromParsedRow(row ParsedCollegeRow) lookupIDs {
	return lookupIDs{
		DegreeLevelIDs: row.DegreeLevelIDs,
		MajorIDs:       row.MajorIDs,
		StudyFormatIDs: row.StudyFormatIDs,
	}
}
