// Package bulkimport implements CSV parsing, validation, and aggregation for
// the admin bulk-upload endpoint (POST /api/v1/universities/bulk).
//
// The parser produces ParsedRow values that mirror the CreateUniversityRequest
// fields but live in this package — that keeps the import graph acyclic
// (bulkimport → university would otherwise cycle). The university service
// converts ParsedRow → CreateUniversityRequest right before insertion.
package bulkimport

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"net/mail"
	"net/url"
	"strconv"
	"strings"

	"fmu-backend/internal/errs"
)

// maxRowErrors caps the per-row error list so a wildly malformed upload
// can't return a megabyte-sized error payload. The handler reports a
// "...and N more" trailer when this fires.
const maxRowErrors = 200

// MaxCSVBytes is the upper bound on the multipart upload body. 20 MiB holds
// ~30k rows of average content with headroom for wide columns; the handler
// wires it into http.MaxBytesReader and returns 413 when exceeded.
const MaxCSVBytes int64 = 20 << 20

// Status values accepted by the bulk endpoint.
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

// ParsedRow is one CSV row that survived parsing. Fields mirror
// CreateUniversityRequest plus a normalized Slug for idempotency checks and
// a LineNumber for error reporting. The university service converts this
// struct into a CreateUniversityRequest right before insertion.
type ParsedRow struct {
	LineNumber int // 1-based row number in the source file (header = 0)
	Slug       string

	Name                     string
	Excerpt                  string
	Overview                 string
	Country                  string
	Continent                string
	State                    string
	City                     string
	FullLocation             string
	Zipcode                  string
	CoverImage               string
	Logo                     string
	InstitutionType          string
	CampusSetting            string
	InStateTuition           float64
	OutOfStateTuition        float64
	InternationalTuition     float64
	TuitionMin               int32
	TuitionMax               int32
	NeedBasedAid             bool
	MeritScholarships        bool
	WorkStudy                bool
	NoApplicationFee         bool
	AcceptanceRate           float64
	TestingPolicy            string
	SatRange                 string
	ActRange                 string
	OnCampusHousing          bool
	FreshmenRequiredOnCampus bool
	ContactEmail             string
	ContactPhone             string
	Website                  string
	AvgHighSchoolGpa         float64
	FoundedYear              int32
	CampusSize               string
	GalleryImages            []string
	IsPopular                bool
	IsFeatured               bool
	MapsUrl                  string
	FullAddress              string
	EmploymentRate           float64
	ResearchOutput           string
	HousingType              string
	SeoTitle                 string
	SeoDescription           string
	Status                   string

	// Lookup names from the CSV (pipe-separated). ResolveNames translates
	// each to UUIDs and writes the result into the corresponding *IDs
	// slice below; the service then passes those UUIDs to the repo.
	DegreeLevelNames []string
	MajorNames       []string
	StudyFormatNames []string
	DegreeLevelIDs   []string
	MajorIDs         []string
	StudyFormatIDs   []string
}

// ParseResult is the output of Parse: every row the parser saw, plus all
// row-level errors collected. The caller decides what to do with errors —
// in the bulk-upload flow any non-empty Errors slice rejects the whole
// upload.
type ParseResult struct {
	Rows   []ParsedRow
	Errors []errs.RowError
}

// Parse reads CSV records from r, validates the header against the known
// column set, decodes each row into a ParsedRow, and aggregates any per-row
// problems. Empty data rows (all fields empty) are skipped silently — the
// admin may have left trailing blank lines in their spreadsheet.
//
// The header row is required. Unknown headers cause a row-0 error and abort
// parsing (the admin should fix the file rather than silently dropping data).
// Headers are matched case-insensitively after trim, and a UTF-8 BOM on the
// first cell is stripped.
func Parse(r io.Reader, status string) ParseResult {
	if status != StatusPublished {
		status = StatusDraft
	}

	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1 // allow variable-width rows; we validate ourselves
	cr.TrimLeadingSpace = true

	out := ParseResult{}

	records, err := cr.ReadAll()
	if err != nil {
		out.Errors = append(out.Errors, errs.RowError{
			Row:     0,
			Column:  "file",
			Message: "file is not valid CSV: " + err.Error(),
		})
		return out
	}
	if len(records) == 0 {
		out.Errors = append(out.Errors, errs.RowError{
			Row:     0,
			Column:  "file",
			Message: "file is empty",
		})
		return out
	}

	header := records[0]
	header = stripBOM(header)
	colIndex, hdrErr := buildHeaderIndex(header)
	if hdrErr != nil {
		out.Errors = append(out.Errors, *hdrErr)
		return out
	}

	// Track slugs seen earlier in the same file so we can reject duplicates
	// with a clear "this row collides with row N" message.
	seenSlugs := map[string]int{}

	for i, record := range records[1:] {
		line := i + 1 // 1-based, header excluded
		if isAllEmpty(record) {
			continue
		}
		row, rowErrs := decodeRow(record, colIndex, line, status)
		if len(rowErrs) > 0 {
			out.Errors = append(out.Errors, rowErrs...)
			continue
		}
		if firstSeen, dup := seenSlugs[row.Slug]; dup {
			out.Errors = append(out.Errors, errs.RowError{
				Row:     line,
				Column:  "slug",
				Value:   row.Slug,
				Message: fmt.Sprintf("duplicate slug within CSV; first occurrence was row %d", firstSeen),
			})
			continue
		}
		seenSlugs[row.Slug] = line
		out.Rows = append(out.Rows, row)

		if len(out.Errors) >= maxRowErrors {
			out.Errors = append(out.Errors, errs.RowError{
				Row:     0,
				Column:  "file",
				Message: fmt.Sprintf("validation aborted after %d row errors; fix and resubmit", len(out.Errors)),
			})
			return out
		}
	}

	return out
}

// stripBOM removes a leading UTF-8 byte-order mark (U+FEFF) from the first
// cell of the header row. encoding/csv doesn't strip it for us; without this
// the first header cell comes through with a leading BOM and never matches.
func stripBOM(cells []string) []string {
	if len(cells) == 0 {
		return cells
	}
	// Built from explicit bytes — Go forbids a literal BOM in source code,
	// so we can't write the string inline.
	var bom strings.Builder
	bom.WriteByte(0xEF)
	bom.WriteByte(0xBB)
	bom.WriteByte(0xBF)
	bomStr := bom.String()
	if strings.HasPrefix(cells[0], bomStr) {
		cells[0] = strings.TrimPrefix(cells[0], bomStr)
	}
	return cells
}

// buildHeaderIndex canonicalizes headers (lowercase, trim) and maps them to
// column positions. Rejects unknown headers with the full list of valid
// columns so the admin can self-correct.
func buildHeaderIndex(header []string) (map[string]int, *errs.RowError) {
	if len(header) == 0 || allEmptyStrings(header) {
		return nil, &errs.RowError{
			Row:     0,
			Column:  "header",
			Message: "header row is empty; expected at least one column",
		}
	}
	idx := make(map[string]int, len(header))
	for i, raw := range header {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		if _, dup := idx[name]; dup {
			return nil, &errs.RowError{
				Row:     0,
				Column:  "header",
				Value:   name,
				Message: fmt.Sprintf("duplicate header column %q", name),
			}
		}
		if _, ok := columnDefs[name]; !ok {
			valid := make([]string, 0, len(columnDefs))
			for k := range columnDefs {
				valid = append(valid, k)
			}
			return nil, &errs.RowError{
				Row:    0,
				Column: "header",
				Value:  name,
				Message: fmt.Sprintf("unknown column %q; valid columns: %s", name, strings.Join(valid, ", ")),
			}
		}
		idx[name] = i
	}
	if _, ok := idx["name"]; !ok {
		return nil, &errs.RowError{
			Row:     0,
			Column:  "header",
			Message: `missing required header "name"`,
		}
	}
	if _, ok := idx["slug"]; !ok {
		return nil, &errs.RowError{
			Row:     0,
			Column:  "header",
			Message: `missing required header "slug"`,
		}
	}
	return idx, nil
}

func allEmptyStrings(cells []string) bool {
	for _, c := range cells {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func isAllEmpty(record []string) bool {
	return allEmptyStrings(record)
}

// decodeRow parses a single CSV record into a ParsedRow, running type
// coercion and per-row validation. Returns zero-or-more RowErrors. Any
// row error means the row is dropped from the result — we never insert a
// row with a known problem.
func decodeRow(record []string, idx map[string]int, line int, status string) (ParsedRow, []errs.RowError) {
	var (
		row   ParsedRow
		errs2 []errs.RowError
	)
	row.LineNumber = line
	row.Status = status

	get := func(col string) string {
		i, ok := idx[col]
		if !ok || i >= len(record) {
			return ""
		}
		return record[i]
	}

	// name (required, always)
	name := strings.TrimSpace(get("name"))
	if name == "" {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "name", Message: "is required"})
	} else if utf8Width(name) > 255 {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "name", Value: name, Message: "must not exceed 255 characters"})
	}
	row.Name = name

	// slug (required, always; normalize)
	rawSlug := strings.TrimSpace(get("slug"))
	slug := strings.ToLower(rawSlug)
	if slug == "" {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "slug", Message: "is required"})
	} else if len(slug) < 2 || len(slug) > 255 {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "slug", Value: rawSlug, Message: "must be between 2 and 255 characters"})
	}
	row.Slug = slug

	// excerpt (optional, max 500)
	row.Excerpt = strings.TrimSpace(get("excerpt"))
	if utf8Width(row.Excerpt) > 500 {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "excerpt", Value: row.Excerpt, Message: "must not exceed 500 characters"})
	}

	// overview (required when status=published)
	row.Overview = strings.TrimSpace(get("overview"))
	if status == StatusPublished && row.Overview == "" {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "overview", Message: "is required when status=published"})
	}

	// location block
	row.Country = strings.TrimSpace(get("country"))
	if status == StatusPublished && row.Country == "" {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "country", Message: "is required when status=published"})
	}
	row.Continent = strings.TrimSpace(get("continent"))
	if utf8Width(row.Continent) > 100 {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "continent", Value: row.Continent, Message: "must not exceed 100 characters"})
	}
	row.State = strings.TrimSpace(get("state"))
	row.City = strings.TrimSpace(get("city"))
	if status == StatusPublished && row.City == "" {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "city", Message: "is required when status=published"})
	}
	row.FullLocation = strings.TrimSpace(get("full_location"))
	row.Zipcode = strings.TrimSpace(get("zipcode"))

	row.Logo = strings.TrimSpace(get("logo"))
	if msg, ok := validateURL(row.Logo, "logo"); !ok {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "logo", Value: row.Logo, Message: msg})
	}
	row.CoverImage = strings.TrimSpace(get("cover_image"))
	if msg, ok := validateURL(row.CoverImage, "cover_image"); !ok {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "cover_image", Value: row.CoverImage, Message: msg})
	}
	row.MapsUrl = strings.TrimSpace(get("maps_url"))
	if msg, ok := validateURL(row.MapsUrl, "maps_url"); !ok {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "maps_url", Value: row.MapsUrl, Message: msg})
	}

	// institution_type, campus_setting (required when published)
	row.InstitutionType = strings.TrimSpace(get("institution_type"))
	if status == StatusPublished && row.InstitutionType == "" {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "institution_type", Message: "is required when status=published"})
	}
	// campus_setting is optional even when published; admins may import
	// a row that hasn't been classified yet and fill it in via the
	// per-university edit endpoint later.
	row.CampusSetting = strings.TrimSpace(get("campus_setting"))
	if utf8Width(row.CampusSetting) > 50 {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "campus_setting", Value: row.CampusSetting, Message: "must not exceed 50 characters"})
	}

	// tuition (numeric, optional)
	for _, col := range []string{"in_state_tuition", "out_of_state_tuition", "international_tuition"} {
		v, msg := parseFloat(get(col))
		if msg != "" {
			errs2 = append(errs2, errs.RowError{Row: line, Column: col, Value: get(col), Message: msg})
		} else {
			switch col {
			case "in_state_tuition":
				row.InStateTuition = v
			case "out_of_state_tuition":
				row.OutOfStateTuition = v
			case "international_tuition":
				row.InternationalTuition = v
			}
		}
	}

	// tuition_min, tuition_max (int, optional, cross-checked)
	tMin, minErr := parseInt(get("tuition_min"))
	tMax, maxErr := parseInt(get("tuition_max"))
	if minErr != "" {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "tuition_min", Value: get("tuition_min"), Message: minErr})
	}
	if maxErr != "" {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "tuition_max", Value: get("tuition_max"), Message: maxErr})
	}
	if minErr == "" && maxErr == "" && tMin != nil && tMax != nil && *tMin > *tMax {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "tuition_min", Message: "must be ≤ tuition_max"})
	}
	if tMin != nil {
		row.TuitionMin = *tMin
	}
	if tMax != nil {
		row.TuitionMax = *tMax
	}

	// booleans (optional, default false)
	for _, col := range []string{
		"need_based_aid", "merit_scholarships", "work_study", "no_application_fee",
		"on_campus_housing", "freshmen_required_on_campus",
		"is_popular", "is_featured",
	} {
		v, msg := parseBool(get(col))
		if msg != "" {
			errs2 = append(errs2, errs.RowError{Row: line, Column: col, Value: get(col), Message: msg})
			continue
		}
		switch col {
		case "need_based_aid":
			row.NeedBasedAid = v
		case "merit_scholarships":
			row.MeritScholarships = v
		case "work_study":
			row.WorkStudy = v
		case "no_application_fee":
			row.NoApplicationFee = v
		case "on_campus_housing":
			row.OnCampusHousing = v
		case "freshmen_required_on_campus":
			row.FreshmenRequiredOnCampus = v
		case "is_popular":
			row.IsPopular = v
		case "is_featured":
			row.IsFeatured = v
		}
	}

	// acceptance_rate, avg_high_school_gpa (float, optional, range-checked)
	if v, msg := parseRangeFloat(get("acceptance_rate"), 0, 100); msg != "" {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "acceptance_rate", Value: get("acceptance_rate"), Message: msg})
	} else if v != nil {
		row.AcceptanceRate = *v
	}
	if v, msg := parseRangeFloat(get("avg_high_school_gpa"), 0, 5); msg != "" {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "avg_high_school_gpa", Value: get("avg_high_school_gpa"), Message: msg})
	} else if v != nil {
		row.AvgHighSchoolGpa = *v
	}
	if v, msg := parseRangeFloat(get("employment_rate"), 0, 100); msg != "" {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "employment_rate", Value: get("employment_rate"), Message: msg})
	} else if v != nil {
		row.EmploymentRate = *v
	}

	// testing_policy, sat_range, act_range, research_output, housing_type
	row.TestingPolicy = strings.TrimSpace(get("testing_policy"))
	row.SatRange = strings.TrimSpace(get("sat_range"))
	row.ActRange = strings.TrimSpace(get("act_range"))
	row.ResearchOutput = strings.TrimSpace(get("research_output"))
	row.HousingType = strings.TrimSpace(get("housing_type"))
	row.CampusSize = strings.TrimSpace(get("campus_size"))
	row.FullAddress = strings.TrimSpace(get("full_address"))
	row.SeoTitle = strings.TrimSpace(get("seo_title"))
	if utf8Width(row.SeoTitle) > 70 {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "seo_title", Value: row.SeoTitle, Message: "must not exceed 70 characters"})
	}
	row.SeoDescription = strings.TrimSpace(get("seo_description"))
	if utf8Width(row.SeoDescription) > 160 {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "seo_description", Value: row.SeoDescription, Message: "must not exceed 160 characters"})
	}

	// contact_email, contact_phone, website
	row.ContactEmail = strings.TrimSpace(get("contact_email"))
	if status == StatusPublished && row.ContactEmail == "" {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "contact_email", Message: "is required when status=published"})
	} else if row.ContactEmail != "" {
		if _, err := mail.ParseAddress(row.ContactEmail); err != nil {
			errs2 = append(errs2, errs.RowError{Row: line, Column: "contact_email", Value: row.ContactEmail, Message: "must be a valid email address"})
		}
	}
	row.ContactPhone = strings.TrimSpace(get("contact_phone"))
	row.Website = strings.TrimSpace(get("website"))
	if status == StatusPublished && row.Website == "" {
		errs2 = append(errs2, errs.RowError{Row: line, Column: "website", Message: "is required when status=published"})
	} else if row.Website != "" {
		if msg, ok := validateURL(row.Website, "website"); !ok {
			errs2 = append(errs2, errs.RowError{Row: line, Column: "website", Value: row.Website, Message: msg})
		}
	}

	// founded_year (1000–2100)
	if raw := strings.TrimSpace(get("founded_year")); raw != "" {
		y, err := strconv.Atoi(raw)
		if err != nil || y < 1000 || y > 2100 {
			errs2 = append(errs2, errs.RowError{Row: line, Column: "founded_year", Value: raw, Message: "must be a year between 1000 and 2100"})
		} else {
			row.FoundedYear = int32(y)
		}
	}

	// gallery_images (pipe-separated URLs)
	if raw := strings.TrimSpace(get("gallery_images")); raw != "" {
		parts := splitPipes(raw)
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if msg, ok := validateURL(p, "gallery_images"); !ok {
				errs2 = append(errs2, errs.RowError{Row: line, Column: "gallery_images", Value: p, Message: msg})
			} else {
				out = append(out, p)
			}
		}
		row.GalleryImages = out
	}

	// multi-value lookups by name (resolution happens in the service layer
	// after a single batched DB round-trip per lookup table)
	// degree_levels / majors / study_formats are optional even when
	// published — an admin may import a row that's still being
	// classified. cleanSplit returns nil for blank cells, so a CSV
	// column that exists with empty values is treated the same as a
	// missing column.
	row.DegreeLevelNames = cleanSplit(get("degree_levels"))
	row.MajorNames = cleanSplit(get("majors"))
	row.StudyFormatNames = cleanSplit(get("study_formats"))

	// Drop the row on any error; never insert a row with a known problem.
	if len(errs2) > 0 {
		return ParsedRow{}, errs2
	}
	return row, nil
}

// ResolveNames translates the human-friendly name lists on each ParsedRow
// into UUID lookups via the provided resolver functions. A row is rejected
// (and the upload aborted) if any of its names don't resolve; the offending
// name is included in the row error message along with up to 10 valid
// alternatives so the admin can self-correct.
//
// nameLookups should map each name (case-insensitive) to its UUID. The
// service layer is responsible for populating these maps from the DB once
// and reusing them for every row.
func ResolveNames(rows []ParsedRow, validDegreeLevels, validMajors, validStudyFormats map[string]string) []errs.RowError {
	var out []errs.RowError
	for i := range rows {
		row := &rows[i]
		ids, errs2 := resolveOne(row.DegreeLevelNames, validDegreeLevels, "degree_levels")
		if len(errs2) > 0 {
			out = append(out, errs2...)
			continue
		}
		row.DegreeLevelIDs = ids

		ids, errs2 = resolveOne(row.MajorNames, validMajors, "majors")
		if len(errs2) > 0 {
			out = append(out, errs2...)
			continue
		}
		row.MajorIDs = ids

		ids, errs2 = resolveOne(row.StudyFormatNames, validStudyFormats, "study_formats")
		if len(errs2) > 0 {
			out = append(out, errs2...)
			continue
		}
		row.StudyFormatIDs = ids
	}
	return out
}

func resolveOne(names []string, lookup map[string]string, column string) ([]string, []errs.RowError) {
	if len(names) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		key := strings.ToLower(strings.TrimSpace(n))
		if id, ok := lookup[key]; ok {
			out = append(out, id)
			continue
		}
		// Surface the first unknown name with a list of valid ones.
		if len(out) == 0 && len(lookup) > 0 {
			samples := make([]string, 0, 10)
			for k := range lookup {
				samples = append(samples, k)
				if len(samples) == 10 {
					break
				}
			}
			return nil, []errs.RowError{{
				Column:  column,
				Value:   n,
				Message: fmt.Sprintf("unknown %s %q; valid examples: %s", strings.TrimSuffix(column, "s"), n, strings.Join(samples, ", ")),
			}}
		}
	}
	return out, nil
}

// splitPipes splits on "|", trims each token, drops empties, and dedupes
// (preserving first-seen order). Returns nil if the input is empty.
func splitPipes(raw string) []string {
	parts := strings.Split(raw, "|")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// cleanSplit is the same as splitPipes but returns nil for any empty input —
// we treat "blank column" as "not provided" rather than "provided an empty
// list".
func cleanSplit(raw string) []string {
	r := strings.TrimSpace(raw)
	if r == "" {
		return nil
	}
	return splitPipes(r)
}

func parseBool(raw string) (bool, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, ""
	}
	switch strings.ToLower(raw) {
	case "true", "t", "y", "yes", "1":
		return true, ""
	case "false", "f", "n", "no", "0":
		return false, ""
	}
	return false, "must be true/false, 1/0, yes/no"
}

func parseInt(raw string) (*int32, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, ""
	}
	v, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return nil, "must be an integer"
	}
	if v > math.MaxInt32 || v < math.MinInt32 {
		return nil, "exceeds integer range"
	}
	out := int32(v)
	return &out, ""
}

func parseFloat(raw string) (float64, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, ""
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, "must be a number"
	}
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return 0, "must be a finite number"
	}
	if v < 0 {
		return 0, "must not be negative"
	}
	return v, ""
}

func parseRangeFloat(raw string, lo, hi float64) (*float64, string) {
	v, msg := parseFloat(raw)
	if msg != "" {
		return nil, msg
	}
	if raw == "" {
		return nil, ""
	}
	if v < lo || v > hi {
		return nil, fmt.Sprintf("must be between %g and %g", lo, hi)
	}
	return &v, ""
}

func validateURL(raw, column string) (string, bool) {
	if raw == "" {
		return "", true
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Sprintf("%s must be a valid URL", column), false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Sprintf("%s must use http or https", column), false
	}
	if u.Host == "" {
		return fmt.Sprintf("%s must include a host", column), false
	}
	return "", true
}

// utf8Width approximates len() while treating runes as one character each
// for the count. Used only for soft length caps where over-by-a-cod is fine.
func utf8Width(s string) int {
	return len([]rune(s))
}

// CapErrorCount caps the supplied RowError slice at maxRowErrors, appending
// a single "and N more" trailer when truncation occurs. Used by the service
// after it has gathered both parser errors and name-resolution errors.
func CapErrorCount(in []errs.RowError) []errs.RowError {
	if len(in) <= maxRowErrors {
		return in
	}
	trailer := errs.RowError{
		Row:     0,
		Column:  "file",
		Message: fmt.Sprintf("...and %d more row error(s); fix and resubmit", len(in)-maxRowErrors),
	}
	return append(in[:maxRowErrors], trailer)
}

// ErrUnknownStatus is returned by ValidateStatus when the caller supplies
// something other than draft|published.
var ErrUnknownStatus = errors.New("status must be 'draft' or 'published'")

// ValidateStatus normalizes the status form field. Empty defaults to draft.
func ValidateStatus(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", StatusDraft:
		return StatusDraft, nil
	case StatusPublished:
		return StatusPublished, nil
	}
	return "", fmt.Errorf("%w (got %q)", ErrUnknownStatus, s)
}

// columnDefs enumerates every CSV header the parser accepts. The set is the
// single source of truth — buildHeaderIndex rejects any header not in this
// map. Add new columns here AND in decodeRow's switch statement (Go can't
// enforce the linkage; the tests catch drift).
var columnDefs = map[string]struct{}{
	// identity
	"name": {}, "slug": {}, "excerpt": {}, "overview": {},
	// location
	"country": {}, "continent": {}, "state": {}, "city": {}, "full_location": {}, "zipcode": {},
	// branding
	"cover_image": {}, "logo": {},
	// classification
	"institution_type": {}, "campus_setting": {},
	// tuition
	"in_state_tuition": {}, "out_of_state_tuition": {}, "international_tuition": {},
	"tuition_min": {}, "tuition_max": {},
	// booleans
	"need_based_aid": {}, "merit_scholarships": {}, "work_study": {}, "no_application_fee": {},
	"on_campus_housing": {}, "freshmen_required_on_campus": {},
	"is_popular": {}, "is_featured": {},
	// academic profile
	"acceptance_rate": {}, "testing_policy": {}, "sat_range": {}, "act_range": {},
	"contact_email": {}, "contact_phone": {}, "website": {},
	"avg_high_school_gpa": {}, "founded_year": {}, "campus_size": {},
	"gallery_images": {},
	// presence
	"maps_url": {}, "full_address": {}, "employment_rate": {},
	"research_output": {}, "housing_type": {},
	// SEO
	"seo_title": {}, "seo_description": {},
	// multi-value name lookups
	"degree_levels": {}, "majors": {}, "study_formats": {},
}