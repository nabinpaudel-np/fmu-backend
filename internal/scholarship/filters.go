package scholarship

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Filters holds the parsed query parameters for GET /api/v1/scholarships.
// Multi-value facet values are DB-shaped (post slug translation) so the
// repository matches lookup names exactly.
type Filters struct {
	EducationLevels []string
	Majors          []string
	Demographics    []string

	ProviderType string
	Country      string
	// Countries is the multi-value form of Country (?countries=US,UK,CA).
	Countries []string
	State     string
	City      string

	// MinGpa is the student's own GPA. A scholarship matches when its required
	// min_gpa is null or <= this value.
	MinGpa *float64

	RequiresFinancialNeed *bool
	EssayRequired         *bool
	// NoRecommendation narrows to scholarships requiring zero recommendation
	// letters (?no_recommendation=true).
	NoRecommendation  *bool
	PortfolioRequired *bool
	IsRenewable       *bool

	AwardMin *float64
	AwardMax *float64

	DeadlineBefore *time.Time
	DeadlineAfter  *time.Time
	// OpenNow narrows to scholarships currently inside their application window.
	OpenNow *bool

	// Status is the lifecycle filter. Empty is interpreted by the handler
	// (forced to "published" for non-admin callers).
	Status     string
	IsPopular  *bool
	IsFeatured *bool
}

func (f Filters) Empty() bool {
	if len(f.EducationLevels)+len(f.Majors)+len(f.Demographics) > 0 {
		return false
	}
	if f.ProviderType != "" || f.Country != "" || f.State != "" || f.City != "" || len(f.Countries) > 0 {
		return false
	}
	if f.MinGpa != nil || f.AwardMin != nil || f.AwardMax != nil {
		return false
	}
	if f.RequiresFinancialNeed != nil || f.EssayRequired != nil || f.NoRecommendation != nil ||
		f.PortfolioRequired != nil || f.IsRenewable != nil {
		return false
	}
	if f.DeadlineBefore != nil || f.DeadlineAfter != nil || f.OpenNow != nil {
		return false
	}
	return f.Status == "" && f.IsPopular == nil && f.IsFeatured == nil
}

func ParseFilters(q url.Values) Filters {
	f := Filters{
		EducationLevels: translateSlugs(q.Get("education_levels"), educationLevelSlugToName),
		Majors:          translateSlugs(q.Get("majors"), majorSlugToName),
		Demographics:    translateSlugs(q.Get("demographics"), demographicSlugToName),
	}
	f.ProviderType = providerTypeSlugToName[q.Get("provider_type")]
	f.Country = q.Get("country")
	f.Countries = splitCSV(q.Get("countries"))
	f.State = q.Get("state")
	f.City = q.Get("city")
	f.Status = normalizeStatus(q.Get("status"))

	f.MinGpa = parseFloatPtr(q.Get("gpa"))
	f.AwardMin = parseFloatPtr(q.Get("award_min"))
	f.AwardMax = parseFloatPtr(q.Get("award_max"))

	f.RequiresFinancialNeed = parseBoolPtr(q.Get("financial_need"))
	f.EssayRequired = parseBoolPtr(q.Get("essay_required"))
	f.NoRecommendation = parseBoolPtr(q.Get("no_recommendation"))
	f.PortfolioRequired = parseBoolPtr(q.Get("portfolio_required"))
	f.IsRenewable = parseBoolPtr(q.Get("renewable"))

	f.DeadlineBefore = parseTimePtr(q.Get("deadline_before"))
	f.DeadlineAfter = parseTimePtr(q.Get("deadline_after"))
	f.OpenNow = parseBoolPtr(q.Get("open_now"))

	f.IsPopular = parseBoolPtr(q.Get("is_popular"))
	f.IsFeatured = parseBoolPtr(q.Get("is_featured"))

	return f
}

// translateSlugs drops unknown values silently — a typo in the URL just narrows
// the result instead of erroring.
func translateSlugs(csv string, table map[string]string) []string {
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if name, ok := table[p]; ok {
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// splitCSV splits a comma-separated query value and trims whitespace. Empty
// entries are dropped; an all-empty input returns nil.
func splitCSV(csv string) []string {
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseFloatPtr(s string) *float64 {
	if s == "" {
		return nil
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &n
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

// parseTimePtr accepts an RFC3339 timestamp or a bare YYYY-MM-DD date. Invalid
// input is treated as "no filter".
func parseTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return &t
	}
	return nil
}

// normalizeStatus maps "all" to "" so the SQL builder treats it as "no filter".
func normalizeStatus(s string) string {
	if s == "all" {
		return ""
	}
	return s
}

// providerTypes is the static Provider Type dropdown, surfaced by /lookups.
var providerTypes = []string{"College/University", "Corporation", "NGO", "Government"}

// Slug → DB-name maps. Keep in sync with
// internal/db/migrations/20260928000002_seed_scholarship_lookup_data.up.sql
// (and, for majors, the university seed migration).

var educationLevelSlugToName = map[string]string{
	"high-school-senior": "High School Senior",
	"undergraduate":      "Undergraduate",
	"graduate":           "Graduate",
	"phd":                "Ph.D.",
}

var demographicSlugToName = map[string]string{
	"first-generation": "First-Generation",
	"women":            "Women",
	"minority":         "Minority / BIPOC",
	"veteran":          "Military / Veteran",
	"lgbtq":            "LGBTQ+",
	"disability":       "Students with Disabilities",
	"international":    "International Students",
}

var majorSlugToName = map[string]string{
	"computer-science": "Computer Science",
	"business":         "Business",
	"engineering":      "Engineering",
	"medicine":         "Medicine",
	"biology":          "Biology",
	"psychology":       "Psychology",
	"economics":        "Economics",
	"art-design":       "Art & Design",
	"law":              "Law",
	"nursing":          "Nursing",
}

var providerTypeSlugToName = map[string]string{
	"college-university": "College/University",
	"corporation":        "Corporation",
	"ngo":                "NGO",
	"government":         "Government",
}
