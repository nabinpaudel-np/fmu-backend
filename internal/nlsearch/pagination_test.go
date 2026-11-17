package nlsearch

import (
	"context"
	"testing"

	"fmu-backend/internal/pagination"
)

// stubRepo returns a fixed number of rows per catalogue.
type stubRepo struct {
	rowsPerEntity int
	calls         []pagination.Query
}

func (s *stubRepo) Search(ctx context.Context, plan *Plan, q pagination.Query) ([]ResultItem, int64, error) {
	s.calls = append(s.calls, q)
	items := make([]ResultItem, 0, s.rowsPerEntity)
	for i := 0; i < s.rowsPerEntity; i++ {
		items = append(items, ResultItem{
			Entity: plan.Entity.Name,
			ID:     string(rune('a' + i)),
			Title:  plan.Entity.Name + " " + string(rune('a'+i)),
		})
	}
	return items, int64(s.rowsPerEntity), nil
}

// stubInterpreter returns a plan naming the given catalogues.
type stubInterpreter struct {
	entities []string
}

func (s *stubInterpreter) Interpret(ctx context.Context, req InterpretRequest) (*FilterSetDTO, error) {
	return &FilterSetDTO{
		Entity:  s.entities[0],
		Filters: []ConditionDTO{},
		Also:    []AlsoDTO{{Entity: s.entities[1]}},
	}, nil
}

func TestService_PagesAcrossCatalogues(t *testing.T) {
	repo := &stubRepo{rowsPerEntity: 5}
	svc := NewService(&stubInterpreter{entities: []string{"universities", "colleges"}}, repo)

	res, err := svc.Search(context.Background(), SearchRequest{Query: "anything", Page: 1, PageSize: 4})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := len(res.Items); got != 4 {
		t.Fatalf("page size: got %d rows, want 4", got)
	}
	if got := res.Meta.Total; got != 10 {
		t.Fatalf("total: got %d, want 10", got)
	}
}

func TestService_SecondPageSkipsRows(t *testing.T) {
	repo := &stubRepo{rowsPerEntity: 5}
	svc := NewService(&stubInterpreter{entities: []string{"universities", "colleges"}}, repo)

	first, err := svc.Search(context.Background(), SearchRequest{Query: "a", Page: 1, PageSize: 4})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	second, err := svc.Search(context.Background(), SearchRequest{Query: "b", Page: 2, PageSize: 4})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(first.Items) != 4 || len(second.Items) != 4 {
		t.Fatalf("expected 4 rows on each page, got %d and %d",
			len(first.Items), len(second.Items))
	}

	// The second page must not repeat the first page's rows.
	seen := map[string]bool{}
	for _, item := range first.Items {
		seen[item.Title] = true
	}
	for _, item := range second.Items {
		if seen[item.Title] {
			t.Fatalf("page two repeated %q from page one", item.Title)
		}
	}
}

func TestService_CatalogueCountIsIndependentOfPage(t *testing.T) {
	repo := &stubRepo{rowsPerEntity: 5}
	svc := NewService(&stubInterpreter{entities: []string{"universities", "colleges"}}, repo)

	res, err := svc.Search(context.Background(), SearchRequest{Query: "a", Page: 2, PageSize: 4})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := res.Meta.Total; got != 10 {
		t.Fatalf("total on page two: got %d, want 10", got)
	}
	if got := res.Meta.TotalPages; got != 3 {
		t.Fatalf("total pages: got %d, want 3", got)
	}
}
