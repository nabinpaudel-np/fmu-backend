package counselling

import (
	"context"

	"fmu-backend/internal/college"
)

// collegeExisterAdapter wraps college.CollegeService so it satisfies the
// slim CollegeExister interface the counselling service depends on.
type collegeExisterAdapter struct {
	svc college.CollegeService
}

// NewCollegeExisterAdapter wraps a concrete college.CollegeService so it
// satisfies the slim CollegeExister interface.
func NewCollegeExisterAdapter(svc college.CollegeService) CollegeExister {
	return &collegeExisterAdapter{svc: svc}
}

func (a *collegeExisterAdapter) GetByID(ctx context.Context, id string) error {
	_, err := a.svc.GetByID(ctx, id)
	return err
}
