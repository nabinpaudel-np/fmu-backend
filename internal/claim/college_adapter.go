package claim

import (
	"context"

	"fmu-backend/internal/college"
)

// collegeExisterAdapter wraps college.CollegeService so it satisfies the
// slim CollegeExister interface the claim service depends on. We only
// need the existence check (and want its errs.ErrNotFound to bubble up),
// not the full CollegeDetailResponse payload.
type collegeExisterAdapter struct {
	svc college.CollegeService
}

// NewCollegeExisterAdapter wraps a concrete college.CollegeService so it
// satisfies the slim CollegeExister interface the claim service depends
// on. The adapter only exposes GetByID (existence check).
func NewCollegeExisterAdapter(svc college.CollegeService) CollegeExister {
	return &collegeExisterAdapter{svc: svc}
}

func (a *collegeExisterAdapter) GetByID(ctx context.Context, id string) error {
	_, err := a.svc.GetByID(ctx, id)
	return err
}
