package claim

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"fmu-backend/internal/db/sqlc"
)

// claimJoinRow is the shared mapping shape used by all claim list/detail
// mappers. Both university and college rows share the same field set (they
// came from the same JOIN shape — just a different target table) so we
// funnel them through one mapper via these adapters.
type claimJoinRow struct {
	Type          ClaimTarget
	ID            string
	TargetID      string
	TargetName    string
	FullName      string
	WorkEmail     string
	Role          string
	DocumentURL   string
	Status        string
	ReviewerID    pgtype.UUID
	ReviewedAt    pgtype.Timestamptz
	ReviewNote    *string
	CreatedUserID pgtype.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func fromUniGetRow(r sqlc.GetUniversityClaimByIDRow) claimJoinRow {
	return claimJoinRow{
		Type:          TargetUniversity,
		ID:            r.ID,
		TargetID:      r.UniversityID,
		TargetName:    r.UniversityName,
		FullName:      r.FullName,
		WorkEmail:     r.WorkEmail,
		Role:          r.Role,
		DocumentURL:   r.DocumentUrl,
		Status:        r.Status,
		ReviewerID:    r.ReviewerID,
		ReviewedAt:    r.ReviewedAt,
		ReviewNote:    r.ReviewNote,
		CreatedUserID: r.CreatedUserID,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func fromUniListRow(r sqlc.ListUniversityClaimsRow) claimJoinRow {
	return claimJoinRow{
		Type:          TargetUniversity,
		ID:            r.ID,
		TargetID:      r.UniversityID,
		TargetName:    r.UniversityName,
		FullName:      r.FullName,
		WorkEmail:     r.WorkEmail,
		Role:          r.Role,
		DocumentURL:   r.DocumentUrl,
		Status:        r.Status,
		ReviewerID:    r.ReviewerID,
		ReviewedAt:    r.ReviewedAt,
		ReviewNote:    r.ReviewNote,
		CreatedUserID: r.CreatedUserID,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func fromColGetRow(r sqlc.GetCollegeClaimByIDRow) claimJoinRow {
	return claimJoinRow{
		Type:          TargetCollege,
		ID:            r.ID,
		TargetID:      r.CollegeID,
		TargetName:    r.CollegeName,
		FullName:      r.FullName,
		WorkEmail:     r.WorkEmail,
		Role:          r.Role,
		DocumentURL:   r.DocumentUrl,
		Status:        r.Status,
		ReviewerID:    r.ReviewerID,
		ReviewedAt:    r.ReviewedAt,
		ReviewNote:    r.ReviewNote,
		CreatedUserID: r.CreatedUserID,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func fromColListRow(r sqlc.ListCollegeClaimsRow) claimJoinRow {
	return claimJoinRow{
		Type:          TargetCollege,
		ID:            r.ID,
		TargetID:      r.CollegeID,
		TargetName:    r.CollegeName,
		FullName:      r.FullName,
		WorkEmail:     r.WorkEmail,
		Role:          r.Role,
		DocumentURL:   r.DocumentUrl,
		Status:        r.Status,
		ReviewerID:    r.ReviewerID,
		ReviewedAt:    r.ReviewedAt,
		ReviewNote:    r.ReviewNote,
		CreatedUserID: r.CreatedUserID,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func toClaimListItem(c claimJoinRow) ClaimListItem {
	return ClaimListItem{
		ID:            c.ID,
		Type:          c.Type,
		TargetID:      c.TargetID,
		TargetName:    c.TargetName,
		FullName:      c.FullName,
		WorkEmail:     c.WorkEmail,
		Role:          c.Role,
		DocumentURL:   c.DocumentURL,
		Status:        c.Status,
		ReviewerID:    uuidStringPtr(c.ReviewerID),
		ReviewedAt:    timePtr(c.ReviewedAt),
		ReviewNote:    c.ReviewNote,
		CreatedUserID: uuidStringPtr(c.CreatedUserID),
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func uuidStringPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	// formatUUID is shared with the user package — inline the canonical
	// 8-4-4-4-12 rendering here to avoid a cross-package helper import.
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
