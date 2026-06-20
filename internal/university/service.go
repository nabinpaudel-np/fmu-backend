package university

import (
	"context"
	"log"
)

type UniversityService interface {
	Create(ctx context.Context, req *CreateUniversityRequest) (*University, error)
}

type universityService struct {
	repo UniversityRepository
}

func NewUniversityService(repo UniversityRepository) UniversityService {
	return &universityService{
		repo: repo,
	}
}

func (s *universityService) Create(ctx context.Context, req *CreateUniversityRequest) (*University, error) {
	uni, err := s.repo.Create(ctx, req)
	if err != nil {
		log.Default().Printf("failed to create university: %v", err)
		return nil, err
	}

	return uni, nil
}
