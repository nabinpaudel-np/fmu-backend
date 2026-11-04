package nlsearch

import (
	"context"
	"errors"
	"testing"

	"fmu-backend/internal/pagination"
)

// fakeInterpreter stands in for the model. Each call returns the next queued
// answer, so a test can make the model fail once and then succeed.
type fakeInterpreter struct {
	entity  string
	answers []*FilterSetDTO
	errs    []error
	calls   []InterpretRequest

	classifyErr error
	classified  int
}

func (f *fakeInterpreter) Classify(_ context.Context, _ string) (*Classification, error) {
	f.classified++
	if f.classifyErr != nil {
		return nil, f.classifyErr
	}
	entity := f.entity
	if entity == "" {
		entity = "universities"
	}
	return &Classification{Entity: entity}, nil
}

func (f *fakeInterpreter) Interpret(_ context.Context, req InterpretRequest) (*FilterSetDTO, error) {
	f.calls = append(f.calls, req)
	i := len(f.calls) - 1
	if i < len(f.errs) && f.errs[i] != nil {
		return nil, f.errs[i]
	}
	if i >= len(f.answers) {
		return nil, ErrProviderFailed
	}
	return f.answers[i], nil
}

// fakeRepository records the plan it was given and returns a fixed result.
type fakeRepository struct {
	plan    *Plan
	items   []ResultItem
	total   int64
	err     error
	called  bool
	lastQty pagination.Query
}

func (f *fakeRepository) Search(_ context.Context, plan *Plan, q pagination.Query) ([]ResultItem, int64, error) {
	f.called = true
	f.plan = plan
	f.lastQty = q
	return f.items, f.total, f.err
}

func TestService_SearchRunsThePlan(t *testing.T) {
	interpreter := &fakeInterpreter{answers: []*FilterSetDTO{{
		Entity:      "universities",
		Filters:     []FilterDTO{{Field: "city", Op: "=", Value: "Kathmandu"}},
		Assumptions: "read the place as the city",
	}}}
	repo := &fakeRepository{
		items: []ResultItem{{Entity: "universities", ID: "u1", Title: "Kathmandu University"}},
		total: 1,
	}
	svc := NewService(interpreter, repo)

	res, err := svc.Search(context.Background(), SearchRequest{Query: "universities in kathmandu"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !repo.called {
		t.Fatal("expected the repository to be queried")
	}
	if repo.plan == nil || repo.plan.Entity.Name != "universities" {
		t.Fatalf("expected a university plan, got %#v", repo.plan)
	}
	if res.Interpretation.Assumptions != "read the place as the city" {
		t.Errorf("expected the assumption to be reported, got %q", res.Interpretation.Assumptions)
	}
	if len(res.Items) != 1 || res.Items[0].Title != "Kathmandu University" {
		t.Errorf("expected the result rows, got %#v", res.Items)
	}
	if res.Meta.Total != 1 || res.Meta.Page != 1 || res.Meta.PageSize != pagination.DefaultPageSize {
		t.Errorf("expected the pagination meta, got %#v", res.Meta)
	}
}

func TestService_RetriesOnceWhenTheModelGetsItWrong(t *testing.T) {
	interpreter := &fakeInterpreter{answers: []*FilterSetDTO{
		{Entity: "universities", Filters: []FilterDTO{{Field: "vibes", Op: "=", Value: "good"}}},
		{Entity: "universities", Filters: []FilterDTO{{Field: "city", Op: "=", Value: "Kathmandu"}}},
	}}
	repo := &fakeRepository{}
	svc := NewService(interpreter, repo)

	res, err := svc.Search(context.Background(), SearchRequest{Query: "universities in kathmandu"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(interpreter.calls) != 2 {
		t.Fatalf("expected the model to be asked twice, got %d calls", len(interpreter.calls))
	}
	if interpreter.calls[1].Correction == "" {
		t.Error("expected the second attempt to carry the correction")
	}
	if len(res.Items) != 0 {
		t.Errorf("expected no rows, got %#v", res.Items)
	}
}

func TestService_ReportsAFilterTheModelCouldNotGetRight(t *testing.T) {
	bad := &FilterSetDTO{
		Entity:  "universities",
		Filters: []FilterDTO{{Field: "vibes", Op: "=", Value: "good"}},
	}
	interpreter := &fakeInterpreter{answers: []*FilterSetDTO{bad, bad}}
	svc := NewService(interpreter, &fakeRepository{})

	_, err := svc.Search(context.Background(), SearchRequest{Query: "universities with good vibes"})
	if err == nil {
		t.Fatal("expected the search to fail")
	}
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected a validation error, got %v", err)
	}
	if len(interpreter.calls) != 2 {
		t.Fatalf("expected exactly one retry, got %d calls", len(interpreter.calls))
	}
}

func TestService_RejectsAnEmptyQuestion(t *testing.T) {
	interpreter := &fakeInterpreter{}
	svc := NewService(interpreter, &fakeRepository{})

	_, err := svc.Search(context.Background(), SearchRequest{Query: "   "})
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected an empty query error, got %v", err)
	}
	if len(interpreter.calls) != 0 {
		t.Fatal("expected the model not to be called for an empty question")
	}
}

func TestService_ReportsWhenNoModelIsConfigured(t *testing.T) {
	svc := NewService(nil, &fakeRepository{})

	_, err := svc.Search(context.Background(), SearchRequest{Query: "universities"})
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("expected the provider to be reported as unavailable, got %v", err)
	}
}

func TestService_CapsThePageSize(t *testing.T) {
	interpreter := &fakeInterpreter{answers: []*FilterSetDTO{{Entity: "universities"}}}
	repo := &fakeRepository{}
	svc := NewService(interpreter, repo)

	if _, err := svc.Search(context.Background(), SearchRequest{Query: "universities", PageSize: 5000}); err != nil {
		t.Fatalf("search: %v", err)
	}
	if repo.lastQty.PageSize != maxPageSize {
		t.Fatalf("expected the page size to be capped at %d, got %d", maxPageSize, repo.lastQty.PageSize)
	}
}

func TestService_SearchesTheClassifiedCatalogue(t *testing.T) {
	interpreter := &fakeInterpreter{
		entity:  "colleges",
		answers: []*FilterSetDTO{{Entity: "colleges"}},
	}
	repo := &fakeRepository{}
	svc := NewService(interpreter, repo)

	if _, err := svc.Search(context.Background(), SearchRequest{Query: "colleges in kathmandu"}); err != nil {
		t.Fatalf("search: %v", err)
	}
	if interpreter.classified != 1 {
		t.Fatalf("expected the catalogue to be chosen once, got %d", interpreter.classified)
	}
	if len(interpreter.calls) != 1 || interpreter.calls[0].Entity != "colleges" {
		t.Fatalf("expected the model to be asked about colleges, got %#v", interpreter.calls)
	}
	if repo.plan == nil || repo.plan.Entity.Name != "colleges" {
		t.Fatalf("expected a college plan, got %#v", repo.plan)
	}
}

func TestService_ReportsAFailedClassification(t *testing.T) {
	interpreter := &fakeInterpreter{classifyErr: ErrProviderFailed}
	svc := NewService(interpreter, &fakeRepository{})

	_, err := svc.Search(context.Background(), SearchRequest{Query: "universities"})
	if !errors.Is(err, ErrProviderFailed) {
		t.Fatalf("expected the provider failure to be reported, got %v", err)
	}
	if len(interpreter.calls) != 0 {
		t.Fatal("expected the model not to be asked for filters after a failed classification")
	}
}
