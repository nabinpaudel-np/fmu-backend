package university

import (
	"net/url"
	"strconv"
	"strings"
)

// Filters holds the parsed query parameters for `GET /api/v1/universities`.
// Values are DB-shaped (post slug translation) so the repository can match
// exact without caring about UX labels.
type Filters struct {
	DegreeLevels        []string
	Majors              []string
	StudyFormats        []string
	SpecialAffiliations []string
	Athletics           []string
	CampusSettings      []string

	InstitutionType string
	TestingPolicy   string
	Country         string
	State           string
	City            string

	TuitionMin    *int
	TuitionMax    *int
	AcceptanceMin *float64
	AcceptanceMax *float64

	NeedBasedAid      *bool
	MeritScholarships *bool
	NoApplicationFee  *bool
	OnCampusHousing   *bool

	HasSupportService map[string]bool
}

func (f Filters) Empty() bool {
	if len(f.DegreeLevels)+len(f.Majors)+len(f.StudyFormats)+
		len(f.SpecialAffiliations)+len(f.Athletics)+len(f.CampusSettings) > 0 {
		return false
	}
	if f.InstitutionType != "" || f.TestingPolicy != "" || f.Country != "" ||
		f.State != "" || f.City != "" {
		return false
	}
	if f.TuitionMin != nil || f.TuitionMax != nil ||
		f.AcceptanceMin != nil || f.AcceptanceMax != nil {
		return false
	}
	if f.NeedBasedAid != nil || f.MeritScholarships != nil ||
		f.NoApplicationFee != nil || f.OnCampusHousing != nil {
		return false
	}
	return len(f.HasSupportService) == 0
}

func ParseFilters(q url.Values) Filters {
	f := Filters{
		DegreeLevels:        translateSlugs(q.Get("degree_levels"), degreeLevelSlugToName),
		Majors:              translateSlugs(q.Get("majors"), majorSlugToName),
		StudyFormats:        translateSlugs(q.Get("study_formats"), studyFormatSlugToName),
		SpecialAffiliations: translateSlugs(q.Get("special_affiliations"), specialAffiliationSlugToName),
		Athletics:           translateSlugs(q.Get("athletics"), athleticSlugToName),
		CampusSettings:      translateSlugs(q.Get("campus_setting"), campusSettingSlugToTitle),
	}
	f.InstitutionType = institutionTypeSlugToName[q.Get("institution_type")]
	f.TestingPolicy = testingPolicySlugToName[q.Get("testing_policy")]
	f.Country = q.Get("country")
	f.State = q.Get("state_province")
	f.City = q.Get("city")
	f.TuitionMin = parseIntPtr(q.Get("tuitionMin"))
	f.TuitionMax = parseIntPtr(q.Get("tuitionMax"))
	f.AcceptanceMin = parseFloatPtr(q.Get("acceptanceMin"))
	f.AcceptanceMax = parseFloatPtr(q.Get("acceptanceMax"))
	f.NeedBasedAid = parseBoolPtr(q.Get("offers_need_based_aid"))
	f.MeritScholarships = parseBoolPtr(q.Get("offers_merit_scholarships"))
	f.NoApplicationFee = parseBoolPtr(q.Get("no_application_fee"))
	f.OnCampusHousing = parseBoolPtr(q.Get("on_campus_housing"))

	// Support services: each ?has_X=true becomes a constraint that the school
	// has that service. Multiple has_* AND together.
	hasSupport := func(param, dbName string) {
		if v := parseBoolPtr(q.Get(param)); v != nil && *v {
			if f.HasSupportService == nil {
				f.HasSupportService = make(map[string]bool)
			}
			f.HasSupportService[dbName] = true
		}
	}
	hasSupport("has_greek_life", "Greek Life")
	hasSupport("has_rotc", "ROTC")
	hasSupport("has_veteran", "Veteran Services")
	hasSupport("has_disability", "Disability Services")
	hasSupport("has_lgbtq", "LGBTQ+ Support")
	hasSupport("has_intl", "International Student Center")

	return f
}

// translateSlugs drops unknown values silently — a typo in the URL just
// narrows the result instead of erroring.
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

func parseIntPtr(s string) *int {
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
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

// Slug → DB-name maps. Keep these in sync with
// internal/db/migrations/20260619000002_seed_university_lookup_data.up.sql.

var degreeLevelSlugToName = map[string]string{
	"certificate": "Certificate",
	"associate":   "Associate",
	"bachelors":   "Bachelor's",
	"masters":     "Master's",
	"doctorate":   "Doctorate",
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

var studyFormatSlugToName = map[string]string{
	"in-person": "100% In-Person",
	"online":    "100% Online",
	"hybrid":    "Hybrid / Blended",
}

var specialAffiliationSlugToName = map[string]string{
	"hbcu":           "HBCU",
	"hsi":            "HSI",
	"womens-college": "Women's College",
	"mens-college":   "Men's College",
}

var athleticSlugToName = map[string]string{
	"ncaa-d1":    "NCAA Division I",
	"ncaa-d2":    "NCAA Division II",
	"ncaa-d3":    "NCAA Division III",
	"naia":       "NAIA",
	"intramural": "Intramural Sports",
}

var campusSettingSlugToTitle = map[string]string{
	"urban":    "Urban",
	"suburban": "Suburban",
	"rural":    "Rural",
}

var institutionTypeSlugToName = map[string]string{
	"public":             "Public",
	"private-nonprofit":  "Private (Non-Profit)",
	"private-for-profit": "Private (For-Profit)",
	"2-year":             "2-Year",
	"4-year":             "4-Year",
}

var testingPolicySlugToName = map[string]string{
	"test-optional": "Optional",
	"test-blind":    "Blind",
	"test-required": "Required",
}
