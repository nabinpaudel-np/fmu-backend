package nlsearch

import (
	"strings"
	"testing"
)

func TestMatchQuality_Affordable(t *testing.T) {
	entity, ok := entityByName("universities")
	if !ok {
		t.Fatal("no universities entity")
	}
	for _, q := range []string{
		"affordable universities",
		"most affordable universities",
		"universities with affordable tuition",
		"cheap universities",
		"budget universities",
		"low cost universities",
		"inexpensive universities",
	} {
		got, ok := MatchQuality(entity, q)
		if !ok {
			t.Errorf("MatchQuality(%q) found no quality", q)
			continue
		}
		if got.Field != "tuition_max" || !got.Sort || !got.Ascending {
			t.Errorf("MatchQuality(%q) = %+v, want a tuition ranking, cheapest first", q, got)
		}
		if got.Threshold {
			t.Errorf("MatchQuality(%q) has a threshold, want a ranking only", q)
		}
	}
}

func TestMatchQuality_SuperlativeIsNotAQuality(t *testing.T) {
	entity, _ := entityByName("universities")
	// "cheapest" is an ordering, not a threshold. It must not match a quality.
	for _, q := range []string{"cheapest universities", "least expensive universities"} {
		if _, ok := MatchQuality(entity, q); ok {
			t.Errorf("MatchQuality(%q) matched a quality, want none", q)
		}
	}
}

func TestMatchQuality_NotOfferedWithoutTheField(t *testing.T) {
	// Scholarships have no tuition field, so "affordable" must not be offered.
	entity, ok := entityByName("scholarships")
	if !ok {
		t.Fatal("no scholarships entity")
	}
	if _, ok := MatchQuality(entity, "affordable scholarships"); ok {
		t.Error("MatchQuality matched a quality for a catalogue with no tuition field")
	}
}

// A money quality ranks rather than filters. Tuition here runs from 0 to
// 9,000,000 across currencies, so a cut-off would drop every university
// whose fees sit above it — which is most of them.
func TestApplyQuality_AffordableOnlyRanks(t *testing.T) {
	entity, _ := entityByName("universities")
	plan := &Plan{Entity: entity}

	ApplyQuality(plan, "affordable universities")

	if len(plan.Filters) != 0 {
		t.Errorf("plan has %d filters, want none for a ranking quality", len(plan.Filters))
	}
	if plan.Sort == nil {
		t.Fatal("plan has no sort, want one ordered by tuition")
	}
	if plan.Sort.Field != "tuition_max" || plan.Sort.Direction != "ascending" {
		t.Errorf("sort = %+v, want tuition_max ascending", plan.Sort)
	}
}

// A quality with a natural line filters as well as ranks.
func TestApplyQuality_SelectiveFiltersAndRanks(t *testing.T) {
	entity, _ := entityByName("universities")
	plan := &Plan{Entity: entity}

	ApplyQuality(plan, "selective universities")

	if len(plan.Filters) != 1 {
		t.Fatalf("plan has %d filters, want 1", len(plan.Filters))
	}
	f := plan.Filters[0]
	if f.Field != "acceptance_rate" || f.Op != OpLt || f.Value.Num != 30 {
		t.Errorf("filter = %+v, want acceptance_rate < 30", f)
	}
	if plan.Sort == nil {
		t.Error("plan has no sort, want one ordered by acceptance rate")
	}
}

func TestApplyQuality_KeepsAnExplicitNumber(t *testing.T) {
	entity, _ := entityByName("universities")
	plan := &Plan{
		Entity: entity,
		Filters: []FilterSpec{
			{Field: "tuition_max", Op: OpLt, Value: Value{Kind: ValueNumber, Num: 10000}},
		},
	}

	ApplyQuality(plan, "affordable universities")

	if len(plan.Filters) != 1 {
		t.Fatalf("plan has %d filters, want the model's own number kept", len(plan.Filters))
	}
	if plan.Filters[0].Value.Num != 10000 {
		t.Errorf("filter value = %v, want the model's 10000", plan.Filters[0].Value.Num)
	}
}

// "expensive" is the mirror of "affordable": most expensive first.
func TestApplyQuality_ExpensiveSortsDescending(t *testing.T) {
	entity, _ := entityByName("universities")
	plan := &Plan{Entity: entity}

	ApplyQuality(plan, "expensive universities")

	if plan.Sort == nil {
		t.Fatal("plan has no sort")
	}
	if plan.Sort.Field != "tuition_min" || plan.Sort.Direction != "descending" {
		t.Errorf("sort = %+v, want tuition_min descending", plan.Sort)
	}
}

func TestApplyQuality_AddsNothingForASuperlative(t *testing.T) {
	entity, _ := entityByName("universities")
	plan := &Plan{Entity: entity}

	ApplyQuality(plan, "cheapest universities")

	if len(plan.Filters) != 0 {
		t.Errorf("plan has %d filters, want none for a superlative", len(plan.Filters))
	}
}

func TestRenderQualities_ShowsTheThreshold(t *testing.T) {
	entity, _ := entityByName("universities")
	var b strings.Builder
	renderQualities(&b, entity)
	got := b.String()
	if got == "" {
		t.Fatal("no qualities rendered")
	}
	if !strings.Contains(got, "affordable") || !strings.Contains(got, "tuition_max") {
		t.Errorf("rendered qualities missing the tuition ranking: %q", got)
	}
	if !strings.Contains(got, "selective") || !strings.Contains(got, "acceptance_rate < 30") {
		t.Errorf("rendered qualities missing the selective threshold: %q", got)
	}
}
