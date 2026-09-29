package scholarship

import (
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"fmu-backend/internal/db/sqlc"
)

func toCreateScholarshipParams(req *CreateScholarshipRequest) sqlc.CreateScholarshipParams {
	return sqlc.CreateScholarshipParams{
		Title:                         req.Title,
		Slug:                          req.Slug,
		Description:                   req.Description,
		AwardAmount:                   req.AwardAmount,
		AwardMin:                      toPgNumeric(req.AwardMin),
		AwardMax:                      toPgNumeric(req.AwardMax),
		Logo:                          &req.Logo,
		NumberOfAwards:                &req.NumberOfAwards,
		IsRenewable:                   req.IsRenewable,
		MinGpa:                        toPgNumeric(req.MinGpa),
		RequiresFinancialNeed:         req.RequiresFinancialNeed,
		Country:                       &req.Country,
		State:                         &req.State,
		City:                          &req.City,
		ApplicationOpenDate:           toPgTimestamptz(req.ApplicationOpenDate),
		ApplicationDeadline:           toPgTimestamptz(req.ApplicationDeadline),
		AwardNotificationDate:         toPgTimestamptz(req.AwardNotificationDate),
		EssayRequired:                 req.EssayRequired,
		EssayPrompt:                   &req.EssayPrompt,
		RecommendationLettersRequired: req.RecommendationLettersRequired,
		TranscriptRequirement:         transcriptOrDefault(req.TranscriptRequirement),
		PortfolioRequired:             req.PortfolioRequired,
		ApplicationUrl:                &req.ApplicationUrl,
		ProviderType:                  &req.ProviderType,
		ProviderName:                  &req.ProviderName,
		ContactEmail:                  &req.ContactEmail,
		ContactPhone:                  &req.ContactPhone,
		InternalNotes:                 &req.InternalNotes,
		UniversityID:                  uuidFromString(req.UniversityID),
		CollegeID:                     uuidFromString(req.CollegeID),
		SeoTitle:                      &req.SeoTitle,
		SeoDescription:                &req.SeoDescription,
		IsPopular:                     req.IsPopular,
		IsFeatured:                    req.IsFeatured,
		Status:                        statusOrDefault(req.Status),
	}
}

func statusOrDefault(s string) string {
	if s == "" {
		return "published"
	}
	return s
}

func transcriptOrDefault(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

func toScholarshipResponse(s sqlc.Scholarship) *ScholarshipResponse {
	return &ScholarshipResponse{
		ID:                            s.ID,
		Title:                         s.Title,
		Slug:                          s.Slug,
		Description:                   s.Description,
		AwardAmount:                   s.AwardAmount,
		AwardMin:                      fromPgNumeric(s.AwardMin),
		AwardMax:                      fromPgNumeric(s.AwardMax),
		Logo:                          derefString(s.Logo),
		NumberOfAwards:                derefInt32(s.NumberOfAwards),
		IsRenewable:                   s.IsRenewable,
		MinGpa:                        fromPgNumeric(s.MinGpa),
		RequiresFinancialNeed:         s.RequiresFinancialNeed,
		Country:                       derefString(s.Country),
		State:                         derefString(s.State),
		City:                          derefString(s.City),
		ApplicationOpenDate:           fromPgTimestamptz(s.ApplicationOpenDate),
		ApplicationDeadline:           fromPgTimestamptz(s.ApplicationDeadline),
		AwardNotificationDate:         fromPgTimestamptz(s.AwardNotificationDate),
		EssayRequired:                 s.EssayRequired,
		EssayPrompt:                   derefString(s.EssayPrompt),
		RecommendationLettersRequired: s.RecommendationLettersRequired,
		TranscriptRequirement:         s.TranscriptRequirement,
		PortfolioRequired:             s.PortfolioRequired,
		ApplicationUrl:                derefString(s.ApplicationUrl),
		ProviderType:                  derefString(s.ProviderType),
		ProviderName:                  derefString(s.ProviderName),
		ContactEmail:                  derefString(s.ContactEmail),
		ContactPhone:                  derefString(s.ContactPhone),
		InternalNotes:                 s.InternalNotes,
		UniversityID:                  uuidStringPtr(s.UniversityID),
		CollegeID:                     uuidStringPtr(s.CollegeID),
		SeoTitle:                      derefString(s.SeoTitle),
		SeoDescription:                derefString(s.SeoDescription),
		IsPopular:                     s.IsPopular,
		IsFeatured:                    s.IsFeatured,
		Status:                        s.Status,
		PublishedAt:                   fromPgTimestamptz(s.PublishedAt),
		CreatedAt:                     s.CreatedAt,
		UpdatedAt:                     s.UpdatedAt,
	}
}

func toScholarshipDetailResponse(
	s sqlc.Scholarship,
	educationLevels []sqlc.EducationLevel,
	majors []sqlc.Major,
	demographics []sqlc.Demographic,
) *ScholarshipDetailResponse {
	return &ScholarshipDetailResponse{
		ScholarshipResponse: *toScholarshipResponse(s),
		EducationLevels: toLookupItems(educationLevels, func(e sqlc.EducationLevel) EducationLevelResponse {
			return EducationLevelResponse{ID: e.ID, Name: e.Name}
		}),
		Majors: toLookupItems(majors, func(m sqlc.Major) MajorResponse {
			return MajorResponse{ID: m.ID, Name: m.Name}
		}),
		Demographics: toLookupItems(demographics, func(d sqlc.Demographic) DemographicResponse {
			return DemographicResponse{ID: d.ID, Name: d.Name}
		}),
	}
}

func toScholarshipSearchResult(r sqlc.SearchScholarshipsRow) ScholarshipSearchResult {
	return ScholarshipSearchResult{
		ID:           r.ID,
		Title:        r.Title,
		Slug:         r.Slug,
		ProviderName: r.ProviderName,
		ProviderType: r.ProviderType,
		Logo:         r.Logo,
		AwardAmount:  r.AwardAmount,
	}
}

func toLookupItems[In any, Out any](items []In, convert func(In) Out) []Out {
	out := make([]Out, len(items))
	for i, item := range items {
		out[i] = convert(item)
	}
	return out
}

// --- pg type + pointer helpers (kept package-local, matching the per-package
// convention used by university/college/claim). ---

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

// toPgNumeric converts a float64 into a pgtype.Numeric. A zero value maps to
// NULL so unset optional numerics (award_min/max, min_gpa) don't get caught by
// range filters. pgtype.Numeric.Scan rejects a float64, so the value is passed
// as its string form.
func toPgNumeric(v float64) pgtype.Numeric {
	if v == 0 {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	_ = n.Scan(strconv.FormatFloat(v, 'f', -1, 64))
	return n
}

// nullableNumeric mirrors toPgNumeric; named separately for use at PATCH call
// sites where the pointer is already known to be non-nil.
func nullableNumeric(v float64) pgtype.Numeric {
	return toPgNumeric(v)
}

func fromPgNumeric(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

func toPgTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func fromPgTimestamptz(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
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

// uuidStringPtr renders a pgtype.UUID as a canonical 8-4-4-4-12 string when
// valid, otherwise nil.
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
