package counselling

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"fmu-backend/internal/db/sqlc"
)

// joinRow is the shared mapping shape used by both the GetByID and List
// query rows. They return identical columns (target_name is added by the
// LEFT JOIN in the SQL) so we funnel both through one mapper.
type joinRow struct {
	ID                  string
	TargetType          *string
	TargetID            pgtype.UUID
	TargetName          string
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
	ResumeUrl           *string
	Status              string
	InquiryType         string
	ReviewerID          pgtype.UUID
	ReviewedAt          pgtype.Timestamptz
	ReviewNote          *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func fromGetRow(r sqlc.GetCounsellingInquiryByIDRow) joinRow {
	return joinRow{
		ID:                  r.ID,
		TargetType:          r.TargetType,
		TargetID:            r.TargetID,
		TargetName:          r.TargetName,
		FullName:            r.FullName,
		Email:               r.Email,
		Phone:               r.Phone,
		Country:             r.Country,
		PreferredUniversity: r.PreferredUniversity,
		ProgramOfInterest:   r.ProgramOfInterest,
		StartTerm:           r.StartTerm,
		CurrentEducation:    r.CurrentEducation,
		TestScores:          r.TestScores,
		Message:             r.Message,
		ResumeUrl:           r.ResumeUrl,
		Status:              r.Status,
		InquiryType:         r.InquiryType,
		ReviewerID:          r.ReviewerID,
		ReviewedAt:          r.ReviewedAt,
		ReviewNote:          r.ReviewNote,
		CreatedAt:           r.CreatedAt,
		UpdatedAt:           r.UpdatedAt,
	}
}

func fromListRow(r sqlc.ListCounsellingInquiriesRow) joinRow {
	return joinRow{
		ID:                  r.ID,
		TargetType:          r.TargetType,
		TargetID:            r.TargetID,
		TargetName:          r.TargetName,
		FullName:            r.FullName,
		Email:               r.Email,
		Phone:               r.Phone,
		Country:             r.Country,
		PreferredUniversity: r.PreferredUniversity,
		ProgramOfInterest:   r.ProgramOfInterest,
		StartTerm:           r.StartTerm,
		CurrentEducation:    r.CurrentEducation,
		TestScores:          r.TestScores,
		Message:             r.Message,
		ResumeUrl:           r.ResumeUrl,
		Status:              r.Status,
		InquiryType:         r.InquiryType,
		ReviewerID:          r.ReviewerID,
		ReviewedAt:          r.ReviewedAt,
		ReviewNote:          r.ReviewNote,
		CreatedAt:           r.CreatedAt,
		UpdatedAt:           r.UpdatedAt,
	}
}

// toListItem flattens the row into the response shape. Nullable columns
// become omitempty JSON fields; target_type maps to the typed enum so the
// frontend can branch on it.
func toListItem(r joinRow) CounsellingListItem {
	return CounsellingListItem{
		ID:                  r.ID,
		Type:                targetTypeFromPtr(r.TargetType),
		InquiryType:         InquiryType(r.InquiryType),
		TargetID:            uuidStringPtr(r.TargetID),
		TargetName:          r.TargetName,
		FullName:            r.FullName,
		Email:               r.Email,
		Phone:               stringPtr(r.Phone),
		Country:             stringPtr(r.Country),
		PreferredUniversity: stringPtr(r.PreferredUniversity),
		ProgramOfInterest:   stringPtr(r.ProgramOfInterest),
		StartTerm:           stringPtr(r.StartTerm),
		CurrentEducation:    stringPtr(r.CurrentEducation),
		TestScores:          stringPtr(r.TestScores),
		Message:             stringPtr(r.Message),
		ResumeURL:           stringPtr(r.ResumeUrl),
		Status:              r.Status,
		ReviewerID:          uuidStringPtr(r.ReviewerID),
		ReviewedAt:          timePtr(r.ReviewedAt),
		ReviewNote:          stringPtr(r.ReviewNote),
		CreatedAt:           r.CreatedAt,
		UpdatedAt:           r.UpdatedAt,
	}
}

// targetTypeFromPtr maps a nullable string column into the TargetType enum.
// NULL → TargetGeneral (general inquiries have target_type=NULL).
func targetTypeFromPtr(p *string) TargetType {
	if p == nil {
		return TargetGeneral
	}
	return TargetType(*p)
}

func stringPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

// uuidStringPtr renders a pgtype.UUID as a canonical 8-4-4-4-12 string when
// valid, otherwise nil. Inlined here to avoid a cross-package helper import.
func uuidStringPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	const digits = "0123456789abcdef"
	out := make([]byte, 36)
	pos := 0
	for i, b := range u.Bytes {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out[pos] = '-'
			pos++
		}
		out[pos] = digits[b>>4]
		out[pos+1] = digits[b&0x0f]
		pos += 2
	}
	s := string(out)
	return &s
}
