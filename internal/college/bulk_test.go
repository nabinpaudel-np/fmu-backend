package college

import (
	"bytes"
	"embed"
	"io"
	"strings"
	"testing"
)

//go:embed fixtures/*.csv
var fixtures embed.FS

// read returns the embedded fixture as an io.Reader.
func read(t *testing.T, name string) io.Reader {
	t.Helper()
	data, err := fixtures.ReadFile("fixtures/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return bytes.NewReader(data)
}

func TestParse_Happy(t *testing.T) {
	res := Parse(read(t, "happy.csv"), StatusPublished)

	if len(res.Errors) != 0 {
		t.Fatalf("expected no errors, got %d: %+v", len(res.Errors), res.Errors)
	}
	if got, want := len(res.Rows), 3; got != want {
		t.Fatalf("expected %d rows, got %d", want, got)
	}
	if got := res.Rows[0].Slug; got != "stanford-engineering" {
		t.Errorf("row 1 slug: got %q, want %q", got, "stanford-engineering")
	}
	if got, want := res.Rows[0].Country, "US"; got != want {
		t.Errorf("row 1 country: got %q, want %q", got, want)
	}
	if got, want := res.Rows[2].Country, "IE"; got != want {
		t.Errorf("row 3 country: got %q, want %q", got, want)
	}
	if got, want := res.Rows[0].DegreeLevelNames[0], "Bachelors"; got != want {
		t.Errorf("row 1 degree_levels[0]: got %q, want %q", got, want)
	}
}

func TestParse_BadEmail(t *testing.T) {
	res := Parse(read(t, "bad_email.csv"), StatusPublished)

	if len(res.Errors) == 0 {
		t.Fatal("expected errors, got none")
	}
	if len(res.Rows) != 2 {
		t.Errorf("expected 2 surviving rows, got %d", len(res.Rows))
	}
	found := false
	for _, e := range res.Errors {
		if e.Row == 2 && e.Column == "contact_email" && e.Value == "not-an-email" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error on row 2 contact_email with value %q, got %+v", "not-an-email", res.Errors)
	}
}

func TestParse_BadLookups(t *testing.T) {
	// Parse only handles the row-level validation; name resolution is a
	// separate step. So bad_lookups.csv parses cleanly (the names are
	// just strings). Then ResolveNames rejects the unknown name.
	res := Parse(read(t, "bad_lookups.csv"), StatusPublished)
	if len(res.Errors) != 0 {
		t.Fatalf("parse should not error on unknown lookups, got %+v", res.Errors)
	}
	if len(res.Rows) != 2 {
		t.Fatalf("expected 2 parsed rows, got %d", len(res.Rows))
	}

	// ResolveNames: only "Bachelors" exists, "Wizardry" doesn't.
	lookups := map[string]string{
		"bachelors": "uuid-bachelors",
		"math":      "uuid-math",
	}
	lookupErrs := ResolveNames(res.Rows, lookups, lookups, nil)
	if len(lookupErrs) == 0 {
		t.Fatal("expected ResolveNames to flag the unknown name, got none")
	}
	found := false
	for _, e := range lookupErrs {
		if e.Column == "degree_levels" && strings.Contains(e.Message, "Wizardry") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error naming Wizardry, got %+v", lookupErrs)
	}
}

func TestParse_DuplicateSlug(t *testing.T) {
	res := Parse(read(t, "duplicate_slug.csv"), StatusPublished)

	if len(res.Errors) == 0 {
		t.Fatal("expected duplicate-slug error, got none")
	}
	if len(res.Rows) != 1 {
		t.Errorf("expected 1 surviving row (first wins), got %d", len(res.Rows))
	}
	found := false
	for _, e := range res.Errors {
		if e.Row == 2 && e.Column == "slug" && strings.Contains(e.Message, "row 1") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected duplicate-slug error referencing row 1, got %+v", res.Errors)
	}
}

func TestParse_BOM(t *testing.T) {
	res := Parse(read(t, "bom.csv"), StatusPublished)

	if len(res.Errors) != 0 {
		t.Fatalf("BOM should be stripped transparently, got %+v", res.Errors)
	}
	if len(res.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(res.Rows))
	}
	if res.Rows[0].Slug != "bom-college" {
		t.Errorf("expected slug=bom-college, got %q", res.Rows[0].Slug)
	}
}

func TestParse_HeaderTypo(t *testing.T) {
	res := Parse(read(t, "header_typo.csv"), StatusPublished)

	if len(res.Errors) == 0 {
		t.Fatal("expected unknown-column error, got none")
	}
	found := false
	for _, e := range res.Errors {
		if e.Row == 0 && e.Column == "header" && strings.Contains(e.Message, "sulg") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected header error mentioning 'sulg', got %+v", res.Errors)
	}
	if len(res.Rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(res.Rows))
	}
}

func TestParse_HeaderOnly(t *testing.T) {
	res := Parse(read(t, "header_only.csv"), StatusPublished)

	// No data rows = 0 rows, 0 errors (no work to do).
	if len(res.Errors) != 0 {
		t.Errorf("expected no errors for header-only file, got %+v", res.Errors)
	}
	if len(res.Rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(res.Rows))
	}
}

func TestParse_EmbeddedNewline(t *testing.T) {
	res := Parse(read(t, "embedded_newline.csv"), StatusPublished)

	if len(res.Errors) != 0 {
		t.Fatalf("quoted embedded newline should parse cleanly, got %+v", res.Errors)
	}
	if len(res.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(res.Rows))
	}
	if !strings.Contains(res.Rows[0].Overview, "embedded newline") {
		t.Errorf("expected overview to contain 'embedded newline', got %q", res.Rows[0].Overview)
	}
}

func TestParse_MissingRequired(t *testing.T) {
	res := Parse(read(t, "missing_required.csv"), StatusPublished)

	// Overview is the only required field when status=published. The
	// fixture row strips it.
	if len(res.Errors) == 0 {
		t.Fatal("expected missing-required error, got none")
	}
	found := false
	for _, e := range res.Errors {
		if e.Column == "overview" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error on column %q, got %+v", "overview", res.Errors)
	}
}

func TestParse_Empty(t *testing.T) {
	res := Parse(strings.NewReader(""), StatusPublished)

	if len(res.Errors) == 0 {
		t.Fatal("expected empty-file error, got none")
	}
	if len(res.Errors) != 1 || res.Errors[0].Column != "file" {
		t.Errorf("expected single file error, got %+v", res.Errors)
	}
}

func TestParse_DraftDoesNotRequirePublishedFields(t *testing.T) {
	// A draft row with only name+slug is valid; the rest can be empty.
	csv := "name,slug\nIncomplete Draft,draft-college\n"
	res := Parse(strings.NewReader(csv), StatusDraft)

	if len(res.Errors) != 0 {
		t.Fatalf("draft should not require published-only fields, got %+v", res.Errors)
	}
	if len(res.Rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(res.Rows))
	}
}

func TestParse_DraftStillValidatesFormats(t *testing.T) {
	// Drafts skip required-fields but format rules still fire (cheap,
	// prevents obvious garbage from entering the DB).
	csv := "name,slug,contact_email\nBad Format,bad-format,not-an-email\n"
	res := Parse(strings.NewReader(csv), StatusDraft)

	if len(res.Errors) == 0 {
		t.Fatal("expected email format error on draft, got none")
	}
}

func TestParse_NoParentColumnAccepted(t *testing.T) {
	// A row that includes a fake parent column header (university_id or
	// university_slug) should be rejected with the standard "unknown
	// column" header error — bulk-uploaded colleges never carry a parent
	// in the CSV.
	csv := "name,slug,university_id\nHas Parent,has-parent,d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b\n"
	res := Parse(strings.NewReader(csv), StatusDraft)

	if len(res.Errors) == 0 {
		t.Fatal("expected header rejection for unknown column 'university_id'")
	}
	found := false
	for _, e := range res.Errors {
		if e.Column == "header" && strings.Contains(e.Message, "university_id") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected header error mentioning 'university_id', got %+v", res.Errors)
	}
}

func TestValidateStatus(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", StatusDraft, false},
		{"draft", StatusDraft, false},
		{"DRAFT", StatusDraft, false},
		{"  draft  ", StatusDraft, false},
		{"published", StatusPublished, false},
		{"PUBLISHED", StatusPublished, false},
		{"bogus", "", true},
	}
	for _, c := range cases {
		got, err := ValidateStatus(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("ValidateStatus(%q) err=%v wantErr=%v", c.in, err, c.wantErr)
		}
		if got != c.want {
			t.Errorf("ValidateStatus(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSplitPipes(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a|b|c", []string{"a", "b", "c"}},
		{" a | b | c ", []string{"a", "b", "c"}},
		{"a||b", []string{"a", "b"}},
		{"a|a|a", []string{"a"}}, // dedupe
	}
	for _, c := range cases {
		got := splitPipes(c.in)
		if !equalSlices(got, c.want) {
			t.Errorf("splitPipes(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}