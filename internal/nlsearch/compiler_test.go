package nlsearch

import (
	"strings"
	"testing"
)

// mustPlan validates a DTO the way the service does, so the compiler tests
// exercise the real path from a model's answer to SQL.
func mustPlan(t *testing.T, dto *FilterSetDTO) *Plan {
	t.Helper()
	plan, verr := Validate(dto)
	if verr != nil {
		t.Fatalf("validate: %v", verr)
	}
	return plan
}

func TestCompile_BindsValuesAndScopesToPublished(t *testing.T) {
	plan := mustPlan(t, &FilterSetDTO{
		Entity:  "universities",
		Filters: []FilterDTO{{Field: "city", Op: "=", Value: "Kathmandu"}},
	})

	compiled := Compile(plan)

	if !strings.Contains(compiled.Where, "lower(u.city) = $1") {
		t.Errorf("expected the city predicate, got %q", compiled.Where)
	}
	if !strings.Contains(compiled.Where, "u.status = 'published'") {
		t.Errorf("expected publication scoping, got %q", compiled.Where)
	}
	if len(compiled.Args) != 1 || compiled.Args[0] != "kathmandu" {
		t.Errorf("expected the value bound as an argument, got %#v", compiled.Args)
	}
}

func TestCompile_ValuesAreNeverInterpolatedIntoSQL(t *testing.T) {
	const injection = "Kathmandu'; DROP TABLE universities; --"

	plan := mustPlan(t, &FilterSetDTO{
		Entity:  "universities",
		Filters: []FilterDTO{{Field: "city", Op: "=", Value: injection}},
	})

	compiled := Compile(plan)

	if strings.Contains(compiled.Where, "DROP TABLE") {
		t.Fatalf("the value leaked into the SQL text: %q", compiled.Where)
	}
	if len(compiled.Args) != 1 || compiled.Args[0] != strings.ToLower(injection) {
		t.Fatalf("expected the value to be bound, got %#v", compiled.Args)
	}
}

func TestCompile_FiltersAreAndEd(t *testing.T) {
	plan := mustPlan(t, &FilterSetDTO{
		Entity: "universities",
		Filters: []FilterDTO{
			{Field: "city", Op: "=", Value: "Kathmandu"},
			{Field: "country", Op: "=", Value: "Nepal"},
		},
	})

	compiled := Compile(plan)

	if !strings.Contains(compiled.Where, " AND ") {
		t.Fatalf("expected the filters to be AND-ed, got %q", compiled.Where)
	}
	if !strings.Contains(compiled.Where, "lower(u.city) = $1") ||
		!strings.Contains(compiled.Where, "lower(u.country) = $2") {
		t.Fatalf("expected both predicates with sequential placeholders, got %q", compiled.Where)
	}
}

func TestCompile_LookupFacetBecomesExists(t *testing.T) {
	plan := mustPlan(t, &FilterSetDTO{
		Entity:  "universities",
		Filters: []FilterDTO{{Field: "majors", Op: "in", Value: []any{"Engineering", "Business"}}},
	})

	compiled := Compile(plan)

	want := "EXISTS (SELECT 1 FROM university_majors jt JOIN majors lk ON lk.id = jt.major_id " +
		"WHERE jt.university_id = u.id AND lower(lk.name) = ANY($1::text[]))"
	if !strings.Contains(compiled.Where, want) {
		t.Fatalf("expected %q, got %q", want, compiled.Where)
	}
	if len(compiled.Args) != 1 {
		t.Fatalf("expected one argument, got %#v", compiled.Args)
	}
	values, ok := compiled.Args[0].([]string)
	if !ok || len(values) != 2 || values[0] != "engineering" || values[1] != "business" {
		t.Fatalf("expected the lower-cased values bound, got %#v", compiled.Args[0])
	}
}

func TestCompile_RelatedEntityBecomesExists(t *testing.T) {
	plan := mustPlan(t, &FilterSetDTO{
		Entity: "universities",
		Related: []RelatedDTO{{
			Entity:  "scholarships",
			Filters: []FilterDTO{{Field: "award_min", Op: ">=", Value: 500000}},
		}},
	})

	compiled := Compile(plan)

	want := "EXISTS (SELECT 1 FROM scholarships s WHERE s.university_id = u.id " +
		"AND s.status = 'published' AND s.award_min >= $1)"
	if !strings.Contains(compiled.Where, want) {
		t.Fatalf("expected %q, got %q", want, compiled.Where)
	}
	if len(compiled.Args) != 1 || compiled.Args[0] != 500000.0 {
		t.Fatalf("expected the amount bound as a number, got %#v", compiled.Args)
	}
}

func TestCompile_MatchesRanksBySimilarity(t *testing.T) {
	plan := mustPlan(t, &FilterSetDTO{
		Entity:  "universities",
		Filters: []FilterDTO{{Field: "name", Op: "matches", Value: "kathamndu"}},
	})

	compiled := Compile(plan)

	if !strings.Contains(compiled.Where, "similarity(u.name, $1) > 0.2") {
		t.Fatalf("expected a fuzzy match, got %q", compiled.Where)
	}
	if !strings.HasPrefix(compiled.OrderBy, "similarity(u.name, $1) DESC") {
		t.Fatalf("expected the closest match first, got %q", compiled.OrderBy)
	}
}

func TestCompile_SortUsesRequestedField(t *testing.T) {
	plan := mustPlan(t, &FilterSetDTO{
		Entity: "universities",
		Sort:   &SortDTO{Field: "tuition_max", Direction: "desc"},
	})

	compiled := Compile(plan)

	if compiled.OrderBy != "u.tuition_max DESC" {
		t.Fatalf("expected the requested ordering, got %q", compiled.OrderBy)
	}
}

func TestCompile_DefaultOrderWhenNoSortIsAsked(t *testing.T) {
	plan := mustPlan(t, &FilterSetDTO{Entity: "scholarships"})

	compiled := Compile(plan)

	if compiled.OrderBy != "s.created_at DESC, s.title ASC" {
		t.Fatalf("expected the default ordering, got %q", compiled.OrderBy)
	}
}

func TestCompile_BooleanAndNumberValuesAreTyped(t *testing.T) {
	plan := mustPlan(t, &FilterSetDTO{
		Entity: "universities",
		Filters: []FilterDTO{
			{Field: "merit_scholarships", Op: "=", Value: "true"},
			{Field: "acceptance_rate", Op: "<=", Value: 50},
		},
	})

	compiled := Compile(plan)

	if !strings.Contains(compiled.Where, "u.merit_scholarships = $1") {
		t.Errorf("expected the boolean predicate, got %q", compiled.Where)
	}
	if !strings.Contains(compiled.Where, "u.acceptance_rate <= $2") {
		t.Errorf("expected the number predicate, got %q", compiled.Where)
	}
	if got, ok := compiled.Args[0].(bool); !ok || !got {
		t.Errorf("expected a boolean argument, got %#v", compiled.Args[0])
	}
	if got, ok := compiled.Args[1].(float64); !ok || got != 50 {
		t.Errorf("expected a number argument, got %#v", compiled.Args[1])
	}
}

// A fuzzy filter inside a related clause ranks the rows of that subquery, not
// the rows being returned. Its column must not reach the outer ORDER BY,
// where the related table is not in scope.
func TestCompile_RelatedFuzzyFilterDoesNotRankTheOuterQuery(t *testing.T) {
	plan := mustPlan(t, &FilterSetDTO{
		Entity: "universities",
		Related: []RelatedDTO{{
			Entity:  "scholarships",
			Filters: []FilterDTO{{Field: "title", Op: "matches", Value: "engineering"}},
		}},
	})

	compiled := Compile(plan)

	if strings.Contains(compiled.OrderBy, "s.title") {
		t.Fatalf("the related table's alias leaked into the ordering: %q", compiled.OrderBy)
	}
	if compiled.OrderBy != "u.name ASC" {
		t.Fatalf("expected the default ordering, got %q", compiled.OrderBy)
	}
}

func TestCompile_OwnFuzzyFilterStillRanksAlongsideARelatedOne(t *testing.T) {
	plan := mustPlan(t, &FilterSetDTO{
		Entity:  "universities",
		Filters: []FilterDTO{{Field: "name", Op: "matches", Value: "kathamndu"}},
		Related: []RelatedDTO{{
			Entity:  "scholarships",
			Filters: []FilterDTO{{Field: "title", Op: "matches", Value: "engineering"}},
		}},
	})

	compiled := Compile(plan)

	if !strings.HasPrefix(compiled.OrderBy, "similarity(u.name, $1) DESC") {
		t.Fatalf("expected the university's own match to rank the results, got %q", compiled.OrderBy)
	}
}

// A wildcard inside a value must be matched literally. Otherwise a question
// about "%" returns every row in the catalogue.
func TestCompile_LikeWildcardsInAValueAreEscaped(t *testing.T) {
	plan := mustPlan(t, &FilterSetDTO{
		Entity:  "universities",
		Filters: []FilterDTO{{Field: "country", Op: "contains", Value: `50%_off\here`}},
	})

	compiled := Compile(plan)

	if !strings.Contains(compiled.Where, `ESCAPE '\'`) {
		t.Fatalf("expected the pattern to declare its escape character, got %q", compiled.Where)
	}
	if len(compiled.Args) != 1 || compiled.Args[0] != `50\%\_off\\here` {
		t.Fatalf("expected the wildcards to be escaped in the bound value, got %#v", compiled.Args)
	}
}

// The trigram score is computed on the text as written, so the raw value is
// scored while the escaped one is only used for the literal match.
func TestCompile_FuzzyMatchScoresTheRawValue(t *testing.T) {
	plan := mustPlan(t, &FilterSetDTO{
		Entity:  "universities",
		Filters: []FilterDTO{{Field: "name", Op: "matches", Value: "kathamndu"}},
	})

	compiled := Compile(plan)

	if len(compiled.Args) != 2 {
		t.Fatalf("expected the raw and the literal value to be bound, got %#v", compiled.Args)
	}
	if compiled.Args[0] != "kathamndu" {
		t.Errorf("expected the scored value to be the raw one, got %#v", compiled.Args[0])
	}
	if compiled.Args[1] != "kathamndu" {
		t.Errorf("expected the literal value to match the raw one here, got %#v", compiled.Args[1])
	}
	if !strings.Contains(compiled.Where, "similarity(u.name, $1) > 0.2") {
		t.Errorf("expected the similarity to use the raw value, got %q", compiled.Where)
	}
}
