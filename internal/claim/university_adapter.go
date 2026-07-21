package claim

import (
	"context"

	"fmu-backend/internal/university"
)

// universityExisterAdapter wraps university.UniversityService so it satisfies
// the slim UniversityExister interface the claim service depends on. We only
// need the existence check (and want its errs.ErrNotFound to bubble up),
// not the full UniversityDetailResponse payload.
type universityExisterAdapter struct {
	svc university.UniversityService
}

// NewUniversityExisterAdapter wraps a concrete university.UniversityService so
// it satisfies the slim UniversityExister interface the claim service depends
// on. The adapter only exposes GetByID (existence check).
func NewUniversityExisterAdapter(svc university.UniversityService) UniversityExister {
	return &universityExisterAdapter{svc: svc}
}

func (a *universityExisterAdapter) GetByID(ctx context.Context, id string) error {
	_, err := a.svc.GetByID(ctx, id)
	return err
}
