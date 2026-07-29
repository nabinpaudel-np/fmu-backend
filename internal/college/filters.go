package college

import "net/url"

// Filters holds the parsed query parameters for `GET /api/v1/colleges`.
// Colleges don't have lookup tables or tuition/acceptance knobs — only the
// fields they actually own go through here, so the parser stays small.
type Filters struct {
	UniversityID string
	Country      string
	State        string
	City         string
	// Status is the lifecycle filter: "draft", "published", "archived",
	// or "all". Empty string is interpreted by the handler (defaults to
	// "published" for non-admin callers).
	Status string
	// IsPopular / IsFeatured: ?is_popular=true and ?is_featured=true narrow
	// to rows with that flag set to true. Omit the param (or pass other
	// values) to leave it unfiltered.
	IsPopular  *bool
	IsFeatured *bool
}

func (f Filters) Empty() bool {
	return f.UniversityID == "" && f.Country == "" && f.State == "" && f.City == "" && f.Status == "" &&
		f.IsPopular == nil && f.IsFeatured == nil
}

func ParseFilters(q url.Values) Filters {
	return Filters{
		UniversityID: q.Get("university_id"),
		Country:      q.Get("country"),
		State:        q.Get("state_province"),
		City:         q.Get("city"),
		Status:       normalizeStatus(q.Get("status")),
		IsPopular:    parseBoolPtr(q.Get("is_popular")),
		IsFeatured:   parseBoolPtr(q.Get("is_featured")),
	}
}

// normalizeStatus maps "all" to "" so the SQL builder's `if f.Status != ""`
// check treats it as "no filter". Empty input stays empty (admin callers
// without ?status= want every row, which the handler already understands).
func normalizeStatus(s string) string {
	if s == "all" {
		return ""
	}
	return s
}

func parseBoolPtr(s string) *bool {
	switch s {
	case "true":
		t := true
		return &t
	case "false":
		f := false
		return &f
	default:
		return nil
	}
}
