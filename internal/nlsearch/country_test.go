package nlsearch

import (
	"strings"
	"testing"
)

// The catalogue stores a country as a full name, so a filter written against
// the ISO code has to be widened into every spelling of that place. Without
// this, "universities in usa" compiles to a comparison against "us" and
// matches nothing.
func TestValidate_CountryBecomesEverySpelling(t *testing.T) {
	dto := &FilterSetDTO{
		Entity: "universities",
		Filters: []ConditionDTO{
			{Field: "country", Op: "=", Value: "USA"},
		},
	}

	plans, verr := Validate(dto)
	if verr != nil {
		t.Fatalf("validate: %v", verr)
	}

	f := plans.Primary.Filters[0]
	if f.Value.Kind != ValueList {
		t.Fatalf("country value is %v, want a list of spellings", f.Value.Kind)
	}
	joined := strings.Join(f.Value.List, "|")
	for _, want := range []string{"usa", "us", "united states of america"} {
		if !strings.Contains(joined, want) {
			t.Errorf("country list %q is missing %q", joined, want)
		}
	}
}

// The stored full name has to be one of the spellings the filter matches.
func TestCompile_CountryMatchesTheStoredName(t *testing.T) {
	dto := &FilterSetDTO{
		Entity: "universities",
		Filters: []ConditionDTO{
			{Field: "country", Op: "=", Value: "USA"},
		},
	}
	plans, verr := Validate(dto)
	if verr != nil {
		t.Fatalf("validate: %v", verr)
	}

	compiled := Compile(plans.Primary)
	if !strings.Contains(compiled.Where, "= ANY(") {
		t.Errorf("country clause is %q, want a set comparison", compiled.Where)
	}
	values, ok := compiled.Args[0].([]string)
	if !ok {
		t.Fatalf("country argument is %T, want []string", compiled.Args[0])
	}
	found := false
	for _, v := range values {
		if v == "united states of america" {
			found = true
		}
	}
	if !found {
		t.Errorf("country spellings %v do not include the stored name", values)
	}
}

func TestValidate_CountryInListBecomesEverySpelling(t *testing.T) {
	dto := &FilterSetDTO{
		Entity: "universities",
		Filters: []ConditionDTO{
			{Field: "country", Op: "in", Value: []any{"USA", "Canada"}},
		},
	}

	plans, verr := Validate(dto)
	if verr != nil {
		t.Fatalf("validate: %v", verr)
	}

	joined := strings.Join(plans.Primary.Filters[0].Value.List, "|")
	for _, want := range []string{"united states of america", "canada"} {
		if !strings.Contains(joined, want) {
			t.Errorf("country list %q is missing %q", joined, want)
		}
	}
}

// A field that is not a country keeps the value it was given.
func TestValidate_PlainFieldIsNotWidened(t *testing.T) {
	dto := &FilterSetDTO{
		Entity: "universities",
		Filters: []ConditionDTO{
			{Field: "city", Op: "=", Value: "Boston"},
		},
	}

	plans, verr := Validate(dto)
	if verr != nil {
		t.Fatalf("validate: %v", verr)
	}

	f := plans.Primary.Filters[0]
	if f.Value.Kind != ValueString || f.Value.Str != "Boston" {
		t.Errorf("city value = %+v, want the string Boston", f.Value)
	}
}
