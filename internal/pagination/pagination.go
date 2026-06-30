package pagination

import (
	"net/http"
	"strconv"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type Query struct {
	Page     int
	PageSize int
}

type Meta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type Response[T any] struct {
	Items []T `json:"items"`
	Meta  Meta `json:"meta"`
}

// ItemsResponse is the non-paginated sibling of Response[T]. It wraps an
// unpaginated slice under the same `items` key so endpoints that return a
// fixed list (e.g. reference/lookup data) share the same shape as paginated
// endpoints.
type ItemsResponse[T any] struct {
	Items []T `json:"items"`
}

func Parse(r *http.Request) Query {
	q := r.URL.Query()
	page := parseIntDefault(q.Get("page"), DefaultPage)
	pageSize := parseIntDefault(q.Get("page_size"), DefaultPageSize)
	if page < 1 {
		page = DefaultPage
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return Query{Page: page, PageSize: pageSize}
}

func (q Query) Offset() int { return (q.Page - 1) * q.PageSize }
func (q Query) Limit() int  { return q.PageSize }

func (q Query) BuildMeta(total int64) Meta {
	totalPages := 0
	if q.PageSize > 0 {
		totalPages = int(total / int64(q.PageSize))
		if total%int64(q.PageSize) > 0 {
			totalPages++
		}
	}
	return Meta{
		Page:       q.Page,
		PageSize:   q.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}