package nlsearch

import (
	"strings"
	"testing"
)

func TestValidate_KeepsSearchTerm(t *testing.T) {
	dto := &FilterSetDTO{
		Entity: "universities",
		Search: "machine learning research",
	}

	plans, verr := Validate(dto)
	if verr != nil {
		t.Fatalf("expected no error, got %v", verr)
	}
	if got, want := plans.Primary.Search, "machine learning research"; got != want {
		t.Fatalf("search term: got %q, want %q", got, want)
	}
}

func TestValidate_SearchTermDroppedWithoutContent(t *testing.T) {
	dto := &FilterSetDTO{
		Entity: "scholarships",
		Search: "machine learning research",
	}

	plans, verr := Validate(dto)
	if verr != nil {
		t.Fatalf("expected no error, got %v", verr)
	}
	if got := plans.Primary.Search; got != "" {
		t.Fatalf("expected the search term to be dropped, got %q", got)
	}
}

func TestCompile_SearchClause(t *testing.T) {
	plan := &Plan{
		Entity: universityEntity,
		Search: "machine learning",
	}

	compiled := Compile(plan)

	if !strings.Contains(compiled.Where, "search_vector @@ websearch_to_tsquery") {
		t.Fatalf("expected a full-text match, got %q", compiled.Where)
	}
	if !strings.Contains(compiled.Where, "u.overview ILIKE") {
		t.Fatalf("expected the fallback match on the prose, got %q", compiled.Where)
	}
	if !strings.Contains(compiled.OrderBy, "ts_rank") {
		t.Fatalf("expected the results to be ranked, got %q", compiled.OrderBy)
	}
	if got, want := compiled.Args[0], "machine learning"; got != want {
		t.Fatalf("first argument: got %v, want %q", got, want)
	}
}

func TestCompile_SearchTermIsNotBoundIntoTheQueryText(t *testing.T) {
	plan := &Plan{
		Entity: universityEntity,
		Search: "'; DROP TABLE universities; --",
	}

	compiled := Compile(plan)

	if strings.Contains(compiled.Where, "DROP TABLE") {
		t.Fatalf("the search term reached the query text: %q", compiled.Where)
	}
	if got, want := compiled.Args[0], "'; DROP TABLE universities; --"; got != want {
		t.Fatalf("first argument: got %v, want %q", got, want)
	}
}

func TestCompile_SearchTermEscapesWildcards(t *testing.T) {
	plan := &Plan{
		Entity: universityEntity,
		Search: "100%",
	}

	compiled := Compile(plan)

	if got, want := compiled.Args[1], "%100\\%%"; got != want {
		t.Fatalf("fallback argument: got %v, want %q", got, want)
	}
}
