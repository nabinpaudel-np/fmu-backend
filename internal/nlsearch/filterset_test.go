package nlsearch

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestValidate_Rejects(t *testing.T) {
	cases := []struct {
		name  string
		dto   *FilterSetDTO
		field string
	}{
		{
			name:  "unknown entity",
			dto:   &FilterSetDTO{Entity: "students"},
			field: "entity",
		},
		{
			name: "unknown field",
			dto: &FilterSetDTO{
				Entity:  "universities",
				Filters: []FilterDTO{{Field: "vibes", Op: "=", Value: "good"}},
			},
			field: "vibes",
		},
		{
			name: "operator not allowed on the field",
			dto: &FilterSetDTO{
				Entity:  "universities",
				Filters: []FilterDTO{{Field: "merit_scholarships", Op: ">", Value: 1}},
			},
			field: "merit_scholarships",
		},
		{
			name: "value outside the closed vocabulary",
			dto: &FilterSetDTO{
				Entity:  "universities",
				Filters: []FilterDTO{{Field: "campus_setting", Op: "=", Value: "Suburban-ish"}},
			},
			field: "campus_setting",
		},
		{
			name: "sorting by a field that is not sortable",
			dto: &FilterSetDTO{
				Entity: "universities",
				Sort:   &SortDTO{Field: "city", Direction: "asc"},
			},
			field: "sort.field",
		},
		{
			name: "unknown sort direction",
			dto: &FilterSetDTO{
				Entity: "universities",
				Sort:   &SortDTO{Field: "name", Direction: "sideways"},
			},
			field: "sort.direction",
		},
		{
			name: "a list on an operator that takes one value",
			dto: &FilterSetDTO{
				Entity:  "universities",
				Filters: []FilterDTO{{Field: "city", Op: "=", Value: []any{"Kathmandu"}}},
			},
			field: "city",
		},
		{
			name: "related entity that does not exist",
			dto: &FilterSetDTO{
				Entity:  "universities",
				Related: []RelatedDTO{{Entity: "colleges"}},
			},
			field: "related.entity",
		},
		{
			name:  "no entity at all",
			dto:   &FilterSetDTO{},
			field: "entity",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, verr := Validate(tc.dto)
			if verr == nil {
				t.Fatalf("expected the filter set to be rejected")
			}
			if verr.Field != tc.field {
				t.Errorf("expected the error to name %q, got %q", tc.field, verr.Field)
			}
		})
	}
}

func TestValidate_Accepts(t *testing.T) {
	cases := []struct {
		name string
		dto  *FilterSetDTO
	}{
		{
			name: "a closed vocabulary value written in another case",
			dto: &FilterSetDTO{
				Entity:  "universities",
				Filters: []FilterDTO{{Field: "campus_setting", Op: "=", Value: "urban"}},
			},
		},
		{
			name: "a number written as a string",
			dto: &FilterSetDTO{
				Entity:  "universities",
				Filters: []FilterDTO{{Field: "acceptance_rate", Op: "<=", Value: "50"}},
			},
		},
		{
			name: "a boolean written as a string",
			dto: &FilterSetDTO{
				Entity:  "universities",
				Filters: []FilterDTO{{Field: "merit_scholarships", Op: "=", Value: "true"}},
			},
		},
		{
			name: "a list on the in operator",
			dto: &FilterSetDTO{
				Entity:  "universities",
				Filters: []FilterDTO{{Field: "country", Op: "in", Value: []any{"Nepal", "India"}}},
			},
		},
		{
			name: "a related scholarship filter",
			dto: &FilterSetDTO{
				Entity: "universities",
				Related: []RelatedDTO{{
					Entity:  "scholarships",
					Filters: []FilterDTO{{Field: "award_min", Op: ">=", Value: 500000}},
				}},
			},
		},
		{
			name: "a fuzzy name match",
			dto: &FilterSetDTO{
				Entity:  "universities",
				Filters: []FilterDTO{{Field: "name", Op: "matches", Value: "kathamndu"}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, verr := Validate(tc.dto); verr != nil {
				t.Fatalf("expected the filter set to be accepted, got %v", verr)
			}
		})
	}
}

func TestValidate_NormalisesValues(t *testing.T) {
	plan, verr := Validate(&FilterSetDTO{
		Entity: "universities",
		Filters: []FilterDTO{
			{Field: "acceptance_rate", Op: "<=", Value: "50"},
			{Field: "merit_scholarships", Op: "=", Value: "true"},
			{Field: "country", Op: "in", Value: []any{"Nepal", " India "}},
		},
	})
	if verr != nil {
		t.Fatalf("validate: %v", verr)
	}

	if plan.Filters[0].Value.Kind != ValueNumber || plan.Filters[0].Value.Num != 50 {
		t.Errorf("expected the number to be coerced, got %#v", plan.Filters[0].Value)
	}
	if plan.Filters[1].Value.Kind != ValueBool || !plan.Filters[1].Value.Bool {
		t.Errorf("expected the boolean to be coerced, got %#v", plan.Filters[1].Value)
	}
	if len(plan.Filters[2].Value.List) != 2 ||
		plan.Filters[2].Value.List[0] != "Nepal" ||
		plan.Filters[2].Value.List[1] != "India" {
		t.Errorf("expected a trimmed list, got %#v", plan.Filters[2].Value.List)
	}
}

func TestValidate_KeepsTheModelsNotes(t *testing.T) {
	plan, verr := Validate(&FilterSetDTO{
		Entity:      "universities",
		Assumptions: "read \"near\" as the same city",
		Unmapped:    []string{"the vibe"},
	})
	if verr != nil {
		t.Fatalf("validate: %v", verr)
	}
	if plan.Assumptions != "read \"near\" as the same city" {
		t.Errorf("expected the assumption to be kept, got %q", plan.Assumptions)
	}
	if len(plan.Unmapped) != 1 || plan.Unmapped[0] != "the vibe" {
		t.Errorf("expected the unmapped part to be kept, got %#v", plan.Unmapped)
	}
}

// The note is rendered to the user, so it is shortened. Shortening must not
// cut a multi-byte character in half and leave broken text behind.
func TestValidate_ShortensANoteWithoutBreakingACharacter(t *testing.T) {
	plan, verr := Validate(&FilterSetDTO{
		Entity:      "universities",
		Assumptions: strings.Repeat("क", 500),
	})
	if verr != nil {
		t.Fatalf("validate: %v", verr)
	}
	if !utf8.ValidString(plan.Assumptions) {
		t.Error("expected the shortened note to be valid text")
	}
	if got := utf8.RuneCountInString(plan.Assumptions); got != maxAssumptionLength {
		t.Errorf("expected %d characters, got %d", maxAssumptionLength, got)
	}
}
