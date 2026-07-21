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
}

func (f Filters) Empty() bool {
	return f.UniversityID == "" && f.Country == "" && f.State == "" && f.City == ""
}

func ParseFilters(q url.Values) Filters {
	return Filters{
		UniversityID: q.Get("university_id"),
		Country:      q.Get("country"),
		State:        q.Get("state_province"),
		City:         q.Get("city"),
	}
}
