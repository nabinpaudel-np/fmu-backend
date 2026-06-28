package university

import (
	"context"
	"log"
)

type UniversityService interface {
	Create(ctx context.Context, req *CreateUniversityRequest) (*CreateUniversityResponse, error)
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
