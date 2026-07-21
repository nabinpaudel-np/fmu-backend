package claim

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"fmu-backend/internal/db/sqlc"
)

type ClaimRepository interface {
	Create(ctx context.Context, universityID, fullName, workEmail, documentURL string) (sqlc.UniversityClaim, error)
	GetByID(ctx context.Context, id string) (sqlc.GetUniversityClaimByIDRow, error)
	List(ctx context.Context, statusFilter string, limit, offset int32) ([]sqlc.ListUniversityClaimsRow, error)
	Count(ctx context.Context, statusFilter string) (int64, error)
	CountPendingForUniversity(ctx context.Context, universityID string) (int64, error)
	Approve(ctx context.Context, claimID, reviewerID string, reviewNote *string, createdUserID string) (sqlc.UniversityClaim, error)
	Reject(ctx context.Context, claimID, reviewerID string, reviewNote *string) (sqlc.UniversityClaim, error)
}

type claimRepository struct {
	queries *sqlc.Queries
}

func NewClaimRepository(queries *sqlc.Queries) ClaimRepository {
	return &claimRepository{queries: queries}
}

func (r *claimRepository) Create(ctx context.Context, universityID, fullName, workEmail, documentURL string) (sqlc.UniversityClaim, error) {
	return r.queries.CreateUniversityClaim(ctx, sqlc.CreateUniversityClaimParams{
		UniversityID: universityID,
		FullName:     fullName,
		WorkEmail:    workEmail,
		DocumentUrl:  documentURL,
	})
}

func (r *claimRepository) GetByID(ctx context.Context, id string) (sqlc.GetUniversityClaimByIDRow, error) {
	row, err := r.queries.GetUniversityClaimByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.GetUniversityClaimByIDRow{}, pgx.ErrNoRows
		}
		return sqlc.GetUniversityClaimByIDRow{}, err
	}
	return row, nil
}

// List returns claim rows filtered by status (empty string = no filter).
func (r *claimRepository) List(ctx context.Context, statusFilter string, limit, offset int32) ([]sqlc.ListUniversityClaimsRow, error) {
	return r.queries.ListUniversityClaims(ctx, sqlc.ListUniversityClaimsParams{
		Column1: statusFilter,
		Limit:   limit,
		Offset:  offset,
	})
}

func (r *claimRepository) Count(ctx context.Context, statusFilter string) (int64, error) {
	return r.queries.CountUniversityClaims(ctx, statusFilter)
}

func (r *claimRepository) CountPendingForUniversity(ctx context.Context, universityID string) (int64, error) {
	return r.queries.CountPendingClaimsForUniversity(ctx, universityID)
}

func (r *claimRepository) Approve(ctx context.Context, claimID, reviewerID string, reviewNote *string, createdUserID string) (sqlc.UniversityClaim, error) {
	return r.queries.ApproveUniversityClaim(ctx, sqlc.ApproveUniversityClaimParams{
		ID:            claimID,
		ReviewerID:    uuidFromString(reviewerID),
		ReviewNote:    reviewNote,
		CreatedUserID: uuidFromString(createdUserID),
	})
}

func (r *claimRepository) Reject(ctx context.Context, claimID, reviewerID string, reviewNote *string) (sqlc.UniversityClaim, error) {
	return r.queries.RejectUniversityClaim(ctx, sqlc.RejectUniversityClaimParams{
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
