package counselling

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"fmu-backend/internal/db/sqlc"
)

// CounsellingRepository wraps the sqlc queries so the service depends on a
// narrow interface and a fake can be swapped in for tests.
type CounsellingRepository interface {
	Create(ctx context.Context, params CreateParams) (sqlc.CounsellingInquiry, error)
	GetByID(ctx context.Context, id string) (sqlc.GetCounsellingInquiryByIDRow, error)
	List(ctx context.Context, params ListParams) ([]sqlc.ListCounsellingInquiriesRow, error)
	Count(ctx context.Context, params ListParams) (int64, error)
	Update(ctx context.Context, params UpdateParams) (sqlc.CounsellingInquiry, error)
}

// CreateParams carries the nullable fields as pointer values so the service
// can pass through empty-string fields untouched (DB enforces NULL vs empty
// string consistently across callers).
type CreateParams struct {
	TargetType          *string
	TargetID            pgtype.UUID
	FullName            string
	Email               string
	Phone               *string
	Country             *string
	PreferredUniversity *string
	ProgramOfInterest   *string
	StartTerm           *string
	CurrentEducation    *string
	TestScores          *string
	Message             *string
	ResumeURL           *string
	InquiryType         string
}

// ListParams groups the list/count filter shape. Both functions take the
// same params so callers can pass them straight through.
type ListParams struct {
	Status      string
	TargetType  string
	TargetID    string
	InquiryType string
	Limit       int32
	Offset      int32
}

// UpdateParams bundles the PATCH payload. ReviewedAt is nullable: pass
// pgtype.Timestamptz{} for "leave alone" and a valid timestamp for "set NOW".
type UpdateParams struct {
	ID         string
	Status     string
	ReviewerID pgtype.UUID
	ReviewedAt pgtype.Timestamptz
	ReviewNote *string
}

type counsellingRepository struct {
	queries *sqlc.Queries
}

func NewCounsellingRepository(queries *sqlc.Queries) CounsellingRepository {
	return &counsellingRepository{queries: queries}
}

func (r *counsellingRepository) Create(ctx context.Context, p CreateParams) (sqlc.CounsellingInquiry, error) {
	return r.queries.CreateCounsellingInquiry(ctx, sqlc.CreateCounsellingInquiryParams{
		TargetType:          p.TargetType,
		TargetID:            p.TargetID,
		FullName:            p.FullName,
		Email:               p.Email,
		Phone:               p.Phone,
		Country:             p.Country,
		PreferredUniversity: p.PreferredUniversity,
		ProgramOfInterest:   p.ProgramOfInterest,
		StartTerm:           p.StartTerm,
		CurrentEducation:    p.CurrentEducation,
		TestScores:          p.TestScores,
		Message:             p.Message,
		ResumeUrl:           p.ResumeURL,
		InquiryType:         p.InquiryType,
	})
}

func (r *counsellingRepository) GetByID(ctx context.Context, id string) (sqlc.GetCounsellingInquiryByIDRow, error) {
	return r.queries.GetCounsellingInquiryByID(ctx, id)
}

func (r *counsellingRepository) List(ctx context.Context, p ListParams) ([]sqlc.ListCounsellingInquiriesRow, error) {
	return r.queries.ListCounsellingInquiries(ctx, sqlc.ListCounsellingInquiriesParams{
		Column1: p.Status,
		Column2: p.TargetType,
		Column3: p.TargetID,
		Column4: p.InquiryType,
		Limit:   p.Limit,
		Offset:  p.Offset,
	})
}

func (r *counsellingRepository) Count(ctx context.Context, p ListParams) (int64, error) {
	return r.queries.CountCounsellingInquiries(ctx, sqlc.CountCounsellingInquiriesParams{
		Column1: p.Status,
		Column2: p.TargetType,
		Column3: p.TargetID,
		Column4: p.InquiryType,
	})
}

func (r *counsellingRepository) Update(ctx context.Context, p UpdateParams) (sqlc.CounsellingInquiry, error) {
	return r.queries.UpdateCounsellingStatus(ctx, sqlc.UpdateCounsellingStatusParams{
		ID:         p.ID,
		Status:     p.Status,
		ReviewerID: p.ReviewerID,
		ReviewedAt: p.ReviewedAt,
		ReviewNote: p.ReviewNote,
	})
}
