package blog

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"fmu-backend/internal/db/sqlc"
)

// formatTimestamptz converts sqlc's nullable pgtype.Timestamptz to an
// RFC3339 string, or returns "" if the column is SQL NULL.
func formatTimestamptz(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}

// formatTimestamp converts a non-nullable time.Time to RFC3339 in UTC.
func formatTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// toResponse renders a sqlc.Blog as the full admin/public response payload.
func toResponse(b sqlc.Blog) *BlogResponse {
	res := &BlogResponse{
		ID:                b.ID,
		Title:             b.Title,
		Slug:              b.Slug,
		MetaDescription:   b.MetaDescription,
		BodyHTML:          b.BodyHtml,
		CoverImage:        b.CoverImage,
		Status:            b.Status,
		AuthorName:        b.AuthorName,
		AuthorDescription: b.AuthorDescription,
		AuthorTitle:       b.AuthorTitle,
		Tags:              normaliseTags(b.Tags),
		CreatedAt:         formatTimestamp(b.CreatedAt),
		UpdatedAt:         formatTimestamp(b.UpdatedAt),
	}
	if pub := formatTimestamptz(b.PublishedAt); pub != "" {
		res.PublishedAt = &pub
	}
	return res
}

// toPublicListItem renders the slim card shape used by the public index.
func toPublicListItem(b sqlc.Blog) PublicListItem {
	return PublicListItem{
		ID:              b.ID,
		Title:           b.Title,
		Slug:            b.Slug,
		MetaDescription: b.MetaDescription,
		CoverImage:      b.CoverImage,
		PublishedAt:     formatTimestamptz(b.PublishedAt),
		Tags:            normaliseTags(b.Tags),
	}
}

// toResponses / toPublicListItems batch versions for list endpoints.
func toResponses(rows []sqlc.Blog) []BlogResponse {
	out := make([]BlogResponse, len(rows))
	for i, r := range rows {
		out[i] = *toResponse(r)
	}
	return out
}

func toPublicListItems(rows []sqlc.Blog) []PublicListItem {
	out := make([]PublicListItem, len(rows))
	for i, r := range rows {
		out[i] = toPublicListItem(r)
	}
	return out
}

// normaliseTags guarantees a non-nil slice so the JSON encoder emits `[]`
// instead of `null` for posts with no tags. The DB already defaults the
// column to '{}', but the driver can hand back nil for an empty array.
func normaliseTags(t []string) []string {
	if t == nil {
		return []string{}
	}
	return t
}