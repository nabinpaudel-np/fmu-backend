package claim

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"fmu-backend/internal/db/sqlc"
)

// UniversityClaimRepository wraps the university-specific sqlc queries so
// the service can depend on a narrow interface (and so swapping in a fake
// for tests is trivial).
type UniversityClaimRepository interface {
	Create(ctx context.Context, universityID, fullName, workEmail, documentURL string) (sqlc.UniversityClaim, error)
	GetByID(ctx context.Context, id string) (sqlc.GetUniversityClaimByIDRow, error)
	List(ctx context.Context, statusFilter string, limit, offset int32) ([]sqlc.ListUniversityClaimsRow, error)
	Count(ctx context.Context, statusFilter string) (int64, error)
	CountPendingForUniversity(ctx context.Context, universityID string) (int64, error)
	Approve(ctx context.Context, claimID, reviewerID string, reviewNote *string, createdUserID string) (sqlc.UniversityClaim, error)
	Reject(ctx context.Context, claimID, reviewerID string, reviewNote *string) (sqlc.UniversityClaim, error)
}

type universityClaimRepository struct {
	queries *sqlc.Queries
}

func NewUniversityClaimRepository(queries *sqlc.Queries) UniversityClaimRepository {
	return &universityClaimRepository{queries: queries}
}

func (r *universityClaimRepository) Create(ctx context.Context, universityID, fullName, workEmail, documentURL string) (sqlc.UniversityClaim, error) {
	return r.queries.CreateUniversityClaim(ctx, sqlc.CreateUniversityClaimParams{
		UniversityID: universityID,
		FullName:     fullName,
		WorkEmail:    workEmail,
		DocumentUrl:  documentURL,
	})
}

func (r *universityClaimRepository) GetByID(ctx context.Context, id string) (sqlc.GetUniversityClaimByIDRow, error) {
	return r.queries.GetUniversityClaimByID(ctx, id)
}

func (r *universityClaimRepository) List(ctx context.Context, statusFilter string, limit, offset int32) ([]sqlc.ListUniversityClaimsRow, error) {
	return r.queries.ListUniversityClaims(ctx, sqlc.ListUniversityClaimsParams{
		Column1: statusFilter,
		Limit:   limit,
		Offset:  offset,
	})
}

func (r *universityClaimRepository) Count(ctx context.Context, statusFilter string) (int64, error) {
	return r.queries.CountUniversityClaims(ctx, statusFilter)
}

func (r *universityClaimRepository) CountPendingForUniversity(ctx context.Context, universityID string) (int64, error) {
	return r.queries.CountPendingClaimsForUniversity(ctx, universityID)
}

func (r *universityClaimRepository) Approve(ctx context.Context, claimID, reviewerID string, reviewNote *string, createdUserID string) (sqlc.UniversityClaim, error) {
	return r.queries.ApproveUniversityClaim(ctx, sqlc.ApproveUniversityClaimParams{
		ID:            claimID,
		ReviewerID:    uuidFromString(reviewerID),
		ReviewNote:    reviewNote,
		CreatedUserID: uuidFromString(createdUserID),
	})
}

func (r *universityClaimRepository) Reject(ctx context.Context, claimID, reviewerID string, reviewNote *string) (sqlc.UniversityClaim, error) {
	return r.queries.RejectUniversityClaim(ctx, sqlc.RejectUniversityClaimParams{
		ID:         claimID,
		ReviewerID: uuidFromString(reviewerID),
		ReviewNote: reviewNote,
	})
}

// CollegeClaimRepository wraps the college-specific sqlc queries. Sibling
// to UniversityClaimRepository.
type CollegeClaimRepository interface {
	Create(ctx context.Context, collegeID, fullName, workEmail, documentURL string) (sqlc.CollegeClaim, error)
	GetByID(ctx context.Context, id string) (sqlc.GetCollegeClaimByIDRow, error)
	List(ctx context.Context, statusFilter string, limit, offset int32) ([]sqlc.ListCollegeClaimsRow, error)
	Count(ctx context.Context, statusFilter string) (int64, error)
	CountPendingForCollege(ctx context.Context, collegeID string) (int64, error)
	Approve(ctx context.Context, claimID, reviewerID string, reviewNote *string, createdUserID string) (sqlc.CollegeClaim, error)
	Reject(ctx context.Context, claimID, reviewerID string, reviewNote *string) (sqlc.CollegeClaim, error)
}

type collegeClaimRepository struct {
	queries *sqlc.Queries
}

func NewCollegeClaimRepository(queries *sqlc.Queries) CollegeClaimRepository {
	return &collegeClaimRepository{queries: queries}
}

func (r *collegeClaimRepository) Create(ctx context.Context, collegeID, fullName, workEmail, documentURL string) (sqlc.CollegeClaim, error) {
	return r.queries.CreateCollegeClaim(ctx, sqlc.CreateCollegeClaimParams{
		CollegeID:   collegeID,
		FullName:    fullName,
		WorkEmail:   workEmail,
		DocumentUrl: documentURL,
	})
}

func (r *collegeClaimRepository) GetByID(ctx context.Context, id string) (sqlc.GetCollegeClaimByIDRow, error) {
	return r.queries.GetCollegeClaimByID(ctx, id)
}

func (r *collegeClaimRepository) List(ctx context.Context, statusFilter string, limit, offset int32) ([]sqlc.ListCollegeClaimsRow, error) {
	return r.queries.ListCollegeClaims(ctx, sqlc.ListCollegeClaimsParams{
		Column1: statusFilter,
		Limit:   limit,
		Offset:  offset,
	})
}

func (r *collegeClaimRepository) Count(ctx context.Context, statusFilter string) (int64, error) {
	return r.queries.CountCollegeClaims(ctx, statusFilter)
}

func (r *collegeClaimRepository) CountPendingForCollege(ctx context.Context, collegeID string) (int64, error) {
	return r.queries.CountPendingClaimsForCollege(ctx, collegeID)
}

func (r *collegeClaimRepository) Approve(ctx context.Context, claimID, reviewerID string, reviewNote *string, createdUserID string) (sqlc.CollegeClaim, error) {
	return r.queries.ApproveCollegeClaim(ctx, sqlc.ApproveCollegeClaimParams{
		ID:            claimID,
		ReviewerID:    uuidFromString(reviewerID),
		ReviewNote:    reviewNote,
		CreatedUserID: uuidFromString(createdUserID),
	})
}

func (r *collegeClaimRepository) Reject(ctx context.Context, claimID, reviewerID string, reviewNote *string) (sqlc.CollegeClaim, error) {
	return r.queries.RejectCollegeClaim(ctx, sqlc.RejectCollegeClaimParams{
		ID:         claimID,
		ReviewerID: uuidFromString(reviewerID),
		ReviewNote: reviewNote,
	})
}

func uuidFromString(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{}
	}
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}
	}
	return u
}
