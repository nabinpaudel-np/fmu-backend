package scholarship

import (
	"context"
	"errors"
	"log"

	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
)

type ScholarshipService interface {
	Create(ctx context.Context, req *CreateScholarshipRequest) (*ScholarshipResponse, error)
	Update(ctx context.Context, id string, req *UpdateScholarshipRequest) (*ScholarshipResponse, error)
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, q pagination.Query, f Filters) ([]ScholarshipListItem, int64, error)
	GetByID(ctx context.Context, id string) (*ScholarshipDetailResponse, error)
	Search(ctx context.Context, q string) ([]ScholarshipSearchResult, error)
	Publish(ctx context.Context, id string) (*ScholarshipResponse, error)
	GetAllLookups(ctx context.Context) (*AllScholarshipLookupsResponse, error)
	GetEducationLevels(ctx context.Context) ([]EducationLevelResponse, error)
	GetDemographics(ctx context.Context) ([]DemographicResponse, error)
}

type scholarshipService struct {
	repo ScholarshipRepository
}

func NewScholarshipService(repo ScholarshipRepository) ScholarshipService {
	return &scholarshipService{repo: repo}
}

func (s *scholarshipService) Create(ctx context.Context, req *CreateScholarshipRequest) (*ScholarshipResponse, error) {
	// Drafts only require title + slug. The CreateScholarshipRequest struct has
	// `validate:"required"` on description/award_amount; for drafts those can be
	// left blank and the publish endpoint re-validates before going public.
	if req.Status == "draft" {
		if err := validateDraftScholarship(req); err != nil {
			return nil, err
		}
	}

	row, err := s.repo.Create(ctx, toCreateScholarshipParams(req), lookupIDs{
		EducationLevelIDs: req.EducationLevelIDs,
		MajorIDs:          req.MajorIDs,
		DemographicIDs:    req.DemographicIDs,
	})
	if err != nil {
		log.Default().Printf("failed to create scholarship: %v", err)
		return nil, err
	}
	return toScholarshipResponse(row), nil
}

func validateDraftScholarship(req *CreateScholarshipRequest) error {
	var missing []string
	if req.Title == "" {
		missing = append(missing, "title")
	}
	if req.Slug == "" {
		missing = append(missing, "slug")
	}
	if len(missing) > 0 {
		return &errs.PublishValidationError{Fields: missing}
	}
	return nil
}

func (s *scholarshipService) Update(ctx context.Context, id string, req *UpdateScholarshipRequest) (*ScholarshipResponse, error) {
	row, err := s.repo.Patch(ctx, id, req)
	if err != nil {
		log.Default().Printf("failed to update scholarship %s: %v", id, err)
		return nil, err
	}
	return toScholarshipResponse(row), nil
}

func (s *scholarshipService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		log.Default().Printf("failed to delete scholarship %s: %v", id, err)
		return err
	}
	return nil
}

func (s *scholarshipService) Get(ctx context.Context, q pagination.Query, f Filters) ([]ScholarshipListItem, int64, error) {
	items, total, err := s.repo.Get(ctx, q, f)
	if err != nil {
		log.Default().Printf("failed to list scholarships: %v", err)
		return nil, 0, err
	}
	return items, total, nil
}

func (s *scholarshipService) GetByID(ctx context.Context, id string) (*ScholarshipDetailResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if !errors.Is(err, errs.ErrNotFound) {
			log.Default().Printf("failed to get scholarship %s: %v", id, err)
		}
		return nil, err
	}

	educationLevels, err := s.repo.GetScholarshipEducationLevels(ctx, id)
	if err != nil {
		log.Default().Printf("failed to load scholarship education levels %s: %v", id, err)
		return nil, err
	}
	majors, err := s.repo.GetScholarshipMajors(ctx, id)
	if err != nil {
		log.Default().Printf("failed to load scholarship majors %s: %v", id, err)
		return nil, err
	}
	demographics, err := s.repo.GetScholarshipDemographics(ctx, id)
	if err != nil {
		log.Default().Printf("failed to load scholarship demographics %s: %v", id, err)
		return nil, err
	}

	return toScholarshipDetailResponse(row, educationLevels, majors, demographics), nil
}

func (s *scholarshipService) Search(ctx context.Context, q string) ([]ScholarshipSearchResult, error) {
	rows, err := s.repo.Search(ctx, q)
	if err != nil {
		log.Default().Printf("failed to search scholarships: %v", err)
		return nil, err
	}
	items := make([]ScholarshipSearchResult, len(rows))
	for i, row := range rows {
		items[i] = toScholarshipSearchResult(row)
	}
	return items, nil
}

func (s *scholarshipService) Publish(ctx context.Context, id string) (*ScholarshipResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if missing := requiredFieldsForScholarshipPublish(row); len(missing) > 0 {
		return nil, &errs.PublishValidationError{Fields: missing}
	}
	published, err := s.repo.Publish(ctx, id)
	if err != nil {
		log.Default().Printf("publish scholarship %s: %v", id, err)
		return nil, err
	}
	return toScholarshipResponse(published), nil
}

// requiredFieldsForScholarshipPublish mirrors the `validate:"required"` tags on
// CreateScholarshipRequest: a public scholarship must have title, slug,
// description, and award_amount.
func requiredFieldsForScholarshipPublish(sc sqlc.Scholarship) []string {
	var missing []string
	if sc.Title == "" {
		missing = append(missing, "title")
	}
	if sc.Slug == "" {
		missing = append(missing, "slug")
	}
	if sc.Description == "" {
		missing = append(missing, "description")
	}
	if sc.AwardAmount == "" {
		missing = append(missing, "award_amount")
	}
	return missing
}

func (s *scholarshipService) GetAllLookups(ctx context.Context) (*AllScholarshipLookupsResponse, error) {
	educationLevels, err := s.repo.GetEducationLevels(ctx)
	if err != nil {
		log.Default().Printf("failed to load education levels: %v", err)
		return nil, err
	}
	demographics, err := s.repo.GetDemographics(ctx)
	if err != nil {
		log.Default().Printf("failed to load demographics: %v", err)
		return nil, err
	}
	majors, err := s.repo.GetMajors(ctx)
	if err != nil {
		log.Default().Printf("failed to load majors: %v", err)
		return nil, err
	}

	return &AllScholarshipLookupsResponse{
		EducationLevels: toLookupItems(educationLevels, func(e sqlc.EducationLevel) EducationLevelResponse {
			return EducationLevelResponse{ID: e.ID, Name: e.Name}
		}),
		Demographics: toLookupItems(demographics, func(d sqlc.Demographic) DemographicResponse {
			return DemographicResponse{ID: d.ID, Name: d.Name}
		}),
		Majors: toLookupItems(majors, func(m sqlc.Major) MajorResponse {
			return MajorResponse{ID: m.ID, Name: m.Name}
		}),
		ProviderTypes: providerTypes,
	}, nil
}

func (s *scholarshipService) GetEducationLevels(ctx context.Context) ([]EducationLevelResponse, error) {
	rows, err := s.repo.GetEducationLevels(ctx)
	if err != nil {
		log.Default().Printf("failed to load education levels: %v", err)
		return nil, err
	}
	return toLookupItems(rows, func(e sqlc.EducationLevel) EducationLevelResponse {
		return EducationLevelResponse{ID: e.ID, Name: e.Name}
	}), nil
}

func (s *scholarshipService) GetDemographics(ctx context.Context) ([]DemographicResponse, error) {
	rows, err := s.repo.GetDemographics(ctx)
	if err != nil {
		log.Default().Printf("failed to load demographics: %v", err)
		return nil, err
	}
	return toLookupItems(rows, func(d sqlc.Demographic) DemographicResponse {
		return DemographicResponse{ID: d.ID, Name: d.Name}
	}), nil
}
