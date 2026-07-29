package university

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
)

type UniversityService interface {
	Create(ctx context.Context, req *CreateUniversityRequest) (*CreateUniversityResponse, error)
	Patch(ctx context.Context, id string, req *PatchUniversityRequest) (*CreateUniversityResponse, error)
	Get(ctx context.Context, q pagination.Query, f Filters) ([]UniversityListItem, int64, error)
	GetByID(ctx context.Context, id string) (*UniversityDetailResponse, error)
	Search(ctx context.Context, q string) ([]UniversitySearchResult, error)
	RepresentedIDs(ctx context.Context, ids []string) (map[string]struct{}, error)
	Stats(ctx context.Context) (*StatsResponse, error)
	Publish(ctx context.Context, id string) (*CreateUniversityResponse, error)
	GetMajors(ctx context.Context) ([]MajorResponse, error)
	GetDegreeLevels(ctx context.Context) ([]DegreeLevelResponse, error)
	GetStudyFormats(ctx context.Context) ([]StudyFormatResponse, error)
	GetSpecialAffiliations(ctx context.Context) ([]SpecialAffiliationResponse, error)
	GetAthletics(ctx context.Context) ([]AthleticResponse, error)
	GetSupportServices(ctx context.Context) ([]SupportServiceResponse, error)
	GetAllLookups(ctx context.Context) (*AllLookupsResponse, error)
}

type universityService struct {
	repo UniversityRepository
}

func NewUniversityService(repo UniversityRepository) UniversityService {
	return &universityService{
		repo: repo,
	}
}

func (s *universityService) Create(ctx context.Context, req *CreateUniversityRequest) (*CreateUniversityResponse, error) {
	// When saving as a draft, only `name` and `slug` are required. The
	// CreateUniversityRequest struct has `validate:"required"` on many other
	// fields, so we run a second, narrower validation pass here for drafts.
	if req.Status == "draft" {
		if err := validateDraftUniversity(req); err != nil {
			return nil, err
		}
	}

	row, err := s.repo.Create(ctx, toCreateUniversityParams(req), lookupIDs{
		DegreeLevelIDs:        req.DegreeLevelIDs,
		MajorIDs:              req.MajorIDs,
		StudyFormatIDs:        req.StudyFormatIDs,
		SpecialAffiliationIDs: req.SpecialAffiliationIDs,
		AthleticIDs:           req.AthleticIDs,
		SupportServiceIDs:     req.SupportServiceIDs,
	})
	if err != nil {
		log.Default().Printf("failed to create university: %v", err)
		return nil, err
	}

	return toCreateUniversityResponse(row), nil
}

// validateDraftUniversity rejects drafts that are missing the only two
// fields a draft must carry (name and slug). All other fields may be left
// blank; the publish endpoint re-validates the full required-field set
// before flipping status to "published".
func validateDraftUniversity(req *CreateUniversityRequest) error {
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

func (s *universityService) Publish(ctx context.Context, id string) (*CreateUniversityResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return nil, errs.ErrNotFound
		}
		log.Default().Printf("get university %s for publish: %v", id, err)
		return nil, err
	}
	if missing := requiredFieldsForPublish(row); len(missing) > 0 {
		return nil, &errs.PublishValidationError{Fields: missing}
	}
	published, err := s.repo.Publish(ctx, id)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		log.Default().Printf("publish university %s: %v", id, err)
		return nil, err
	}
	return toCreateUniversityResponse(published), nil
}

// requiredFieldsForPublish mirrors the `validate:"required"` tags on
// CreateUniversityRequest. Mirroring (not reusing the validator) keeps the
// publish check robust against future DTO tweaks that drop a required tag —
// the publish check stays tied to "this row must be presentable to the
// public".
func requiredFieldsForPublish(u sqlc.University) []string {
	var missing []string
	if u.Name == "" {
		missing = append(missing, "name")
	}
	if u.Slug == "" {
		missing = append(missing, "slug")
	}
	if deref(u.Overview) == "" {
		missing = append(missing, "overview")
	}
	if deref(u.Country) == "" {
		missing = append(missing, "country")
	}
	if deref(u.City) == "" {
		missing = append(missing, "city")
	}
	if deref(u.InstitutionType) == "" {
		missing = append(missing, "institution_type")
	}
	if deref(u.CampusSetting) == "" {
		missing = append(missing, "campus_setting")
	}
	if deref(u.ContactEmail) == "" {
		missing = append(missing, "contact_email")
	}
	if deref(u.Website) == "" {
		missing = append(missing, "website")
	}
	return missing
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (s *universityService) Patch(ctx context.Context, id string, req *PatchUniversityRequest) (*CreateUniversityResponse, error) {
	if claims, err := auth.ClaimsFromContext(ctx); err == nil && claims.Role == auth.RoleRepresentative {
		if req.Name != nil || req.Slug != nil {
			return nil, errs.ErrRepCannotChangeNameOrSlug
		}
	}

	row, err := s.repo.Patch(ctx, id, req)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		log.Default().Printf("failed to patch university %s: %v", id, err)
		return nil, err
	}

	return toCreateUniversityResponse(row), nil
}

func (s *universityService) Get(ctx context.Context, q pagination.Query, f Filters) ([]UniversityListItem, int64, error) {
	rows, total, err := s.repo.Get(ctx, q, f)
	if err != nil {
		log.Default().Printf("failed to list universities: %v", err)
		return nil, 0, err
	}
	items := make([]UniversityListItem, len(rows))
	for i, row := range rows {
		items[i] = toUniversityListItem(row)
	}
	return items, total, nil
}

func (s *universityService) GetByID(ctx context.Context, id string) (*UniversityDetailResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound
		}
		// Bad UUIDs are rejected by the cast before any row scan, so we
		// get 22P02 (invalid_text_representation) instead of ErrNoRows.
		// Treat both as not-found so the handler returns 404.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return nil, errs.ErrNotFound
		}
		log.Default().Printf("failed to get university %s: %v", id, err)
		return nil, err
	}

	degreeLevels, err := s.repo.GetUniversityDegreeLevels(ctx, id)
	if err != nil {
		return nil, err
	}
	majors, err := s.repo.GetUniversityMajors(ctx, id)
	if err != nil {
		return nil, err
	}
	studyFormats, err := s.repo.GetUniversityStudyFormats(ctx, id)
	if err != nil {
		return nil, err
	}
	specialAffiliations, err := s.repo.GetUniversitySpecialAffiliations(ctx, id)
	if err != nil {
		return nil, err
	}
	athletics, err := s.repo.GetUniversityAthletics(ctx, id)
	if err != nil {
		return nil, err
	}
	supportServices, err := s.repo.GetUniversitySupportServices(ctx, id)
	if err != nil {
		return nil, err
	}

	return toUniversityDetailResponse(row, degreeLevels, majors, studyFormats, specialAffiliations, athletics, supportServices), nil
}

func (s *universityService) GetMajors(ctx context.Context) ([]MajorResponse, error) {
	rows, err := s.repo.GetMajors(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]MajorResponse, len(rows))
	for i, row := range rows {
		out[i] = MajorResponse{ID: row.ID, Name: row.Name}
	}
	return out, nil
}

func (s *universityService) GetDegreeLevels(ctx context.Context) ([]DegreeLevelResponse, error) {
	rows, err := s.repo.GetDegreeLevels(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]DegreeLevelResponse, len(rows))
	for i, row := range rows {
		out[i] = DegreeLevelResponse{ID: row.ID, Name: row.Name}
	}
	return out, nil
}

func (s *universityService) GetStudyFormats(ctx context.Context) ([]StudyFormatResponse, error) {
	rows, err := s.repo.GetStudyFormats(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]StudyFormatResponse, len(rows))
	for i, row := range rows {
		out[i] = StudyFormatResponse{ID: row.ID, Name: row.Name}
	}
	return out, nil
}

func (s *universityService) GetSpecialAffiliations(ctx context.Context) ([]SpecialAffiliationResponse, error) {
	rows, err := s.repo.GetSpecialAffiliations(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SpecialAffiliationResponse, len(rows))
	for i, row := range rows {
		out[i] = SpecialAffiliationResponse{ID: row.ID, Name: row.Name}
	}
	return out, nil
}

func (s *universityService) GetAthletics(ctx context.Context) ([]AthleticResponse, error) {
	rows, err := s.repo.GetAthletics(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AthleticResponse, len(rows))
	for i, row := range rows {
		out[i] = AthleticResponse{ID: row.ID, Name: row.Name}
	}
	return out, nil
}

func (s *universityService) GetSupportServices(ctx context.Context) ([]SupportServiceResponse, error) {
	rows, err := s.repo.GetSupportServices(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SupportServiceResponse, len(rows))
	for i, row := range rows {
		out[i] = SupportServiceResponse{ID: row.ID, Name: row.Name}
	}
	return out, nil
}

func (s *universityService) Search(ctx context.Context, q string) ([]UniversitySearchResult, error) {
	rows, err := s.repo.Search(ctx, q)
	if err != nil {
		log.Default().Printf("search universities q=%q: %v", q, err)
		return nil, err
	}
	items := make([]UniversitySearchResult, len(rows))
	for i, row := range rows {
		items[i] = toUniversitySearchResult(row)
	}
	return items, nil
}

func (s *universityService) RepresentedIDs(ctx context.Context, ids []string) (map[string]struct{}, error) {
	set, err := s.repo.RepresentedIDs(ctx, ids)
	if err != nil {
		log.Default().Printf("failed to list represented university ids: %v", err)
		return nil, err
	}
	return set, nil
}

func (s *universityService) Stats(ctx context.Context) (*StatsResponse, error) {
	stats, err := s.repo.Stats(ctx)
	if err != nil {
		log.Default().Printf("stats universities: %v", err)
		return nil, err
	}
	return &StatsResponse{
		TotalUniversities: stats.TotalUniversities,
		TotalCountries:    stats.TotalCountries,
		TotalFeatured:     stats.TotalFeatured,
		TotalPopular:      stats.TotalPopular,
	}, nil
}

func (s *universityService) GetAllLookups(ctx context.Context) (*AllLookupsResponse, error) {
	majors, err := s.GetMajors(ctx)
	if err != nil {
		return nil, err
	}
	degreeLevels, err := s.GetDegreeLevels(ctx)
	if err != nil {
		return nil, err
	}
	studyFormats, err := s.GetStudyFormats(ctx)
	if err != nil {
		return nil, err
	}
	specialAffiliations, err := s.GetSpecialAffiliations(ctx)
	if err != nil {
		return nil, err
	}
	athletics, err := s.GetAthletics(ctx)
	if err != nil {
		return nil, err
	}
	supportServices, err := s.GetSupportServices(ctx)
	if err != nil {
		return nil, err
	}
	return &AllLookupsResponse{
		Majors:              majors,
		DegreeLevels:        degreeLevels,
		StudyFormats:        studyFormats,
		SpecialAffiliations: specialAffiliations,
		Athletics:           athletics,
		SupportServices:     supportServices,
	}, nil
}
