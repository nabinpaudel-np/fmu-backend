package counselling

import (
	"context"

	"fmu-backend/internal/university"
)

// universityExisterAdapter wraps university.UniversityService so it satisfies
// the slim UniversityExister interface the counselling service depends on.
// We only need the existence check (and want its errs.ErrNotFound to bubble
// up), not the full detail payload.
type universityExisterAdapter struct {
	svc university.UniversityService
}

// NewUniversityExisterAdapter wraps a concrete university.UniversityService
// so it satisfies the slim UniversityExister interface.
func NewUniversityExisterAdapter(svc university.UniversityService) UniversityExister {
	return &universityExisterAdapter{svc: svc}
}

func (a *universityExisterAdapter) GetByID(ctx context.Context, id string) error {
	_, err := a.svc.GetByID(ctx, id)
	return err
}
