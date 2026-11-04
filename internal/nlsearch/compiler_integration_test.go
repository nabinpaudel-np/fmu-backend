package nlsearch

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"fmu-backend/internal/pagination"
)

// These tests run the compiled SQL against a real database, which is the only
// way to be sure the compiler's joins, casts and junction tables match the
// schema. They are skipped unless a database URL is provided:
//
//	NLSEARCH_TEST_DATABASE_URL=postgres://user:pass@localhost:5432/fmu-backend go test ./internal/nlsearch/...

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("NLSEARCH_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set NLSEARCH_TEST_DATABASE_URL to run the database tests")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestRepository_SearchRunsAgainstTheDatabase(t *testing.T) {
	repo := NewRepository(testPool(t))
	ctx := context.Background()

	cases := []struct {
		name string
		dto  *FilterSetDTO
	}{
		{
			name: "a university with a place and a field of study",
			dto: &FilterSetDTO{
				Entity: "universities",
				Filters: []FilterDTO{
					{Field: "country", Op: "=", Value: "Nepal"},
					{Field: "majors", Op: "in", Value: []any{"Engineering"}},
				},
			},
		},
		{
			name: "a university with a scholarship attached",
			dto: &FilterSetDTO{
				Entity: "universities",
				Related: []RelatedDTO{{
					Entity:  "scholarships",
					Filters: []FilterDTO{{Field: "award_min", Op: ">=", Value: 1000}},
				}},
			},
		},
		{
			name: "a fuzzy name match",
			dto: &FilterSetDTO{
				Entity:  "universities",
				Filters: []FilterDTO{{Field: "name", Op: "matches", Value: "university"}},
			},
		},
		{
			name: "a college with a boolean filter",
			dto: &FilterSetDTO{
				Entity:  "colleges",
				Filters: []FilterDTO{{Field: "is_featured", Op: "=", Value: true}},
			},
		},
		{
			name: "a scholarship with a facet and an order",
			dto: &FilterSetDTO{
				Entity:  "scholarships",
				Filters: []FilterDTO{{Field: "education_levels", Op: "in", Value: []any{"Undergraduate"}}},
				Sort:    &SortDTO{Field: "award_max", Direction: "desc"},
			},
		},
		{
			// A fuzzy filter inside a related clause must not reach the
			// outer ORDER BY, where the related table is out of scope.
			name: "a university matched by a fuzzy scholarship title",
			dto: &FilterSetDTO{
				Entity: "universities",
				Related: []RelatedDTO{{
					Entity:  "scholarships",
					Filters: []FilterDTO{{Field: "title", Op: "matches", Value: "scholarship"}},
				}},
			},
		},
		{
			// A wildcard in a value is matched literally, so this returns
			// the rows containing a percent sign rather than every row.
			name: "a university whose country contains a percent sign",
			dto: &FilterSetDTO{
				Entity:  "universities",
				Filters: []FilterDTO{{Field: "country", Op: "contains", Value: "%"}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, verr := Validate(tc.dto)
			if verr != nil {
				t.Fatalf("validate: %v", verr)
			}
			items, total, err := repo.Search(ctx, plan, pagination.Query{Page: 1, PageSize: 10})
			if err != nil {
				t.Fatalf("search: %v", err)
			}
			if total < 0 {
				t.Errorf("expected a non-negative total, got %d", total)
			}
			if len(items) > 10 {
				t.Errorf("expected at most 10 rows, got %d", len(items))
			}
			for _, item := range items {
				if item.Entity != plan.Entity.Name {
					t.Errorf("expected the row to be tagged %q, got %q", plan.Entity.Name, item.Entity)
				}
				if item.ID == "" {
					t.Error("expected the row to carry an id")
				}
			}
		})
	}
}

// scriptedInterpreter answers with a fixed filter set, standing in for the
// model so the whole flow can be run against a real database.
type scriptedInterpreter struct {
	entity string
	dto    *FilterSetDTO
}

func (s *scriptedInterpreter) Classify(_ context.Context, _ string) (*Classification, error) {
	return &Classification{Entity: s.entity}, nil
}

func (s *scriptedInterpreter) Interpret(_ context.Context, _ InterpretRequest) (*FilterSetDTO, error) {
	dto := *s.dto
	return &dto, nil
}

// The whole flow, from a question to rows: the model's answer is validated,
// compiled and run, and the answer comes back with the interpretation the
// caller needs to show the user.
func TestService_SearchRunsAgainstTheDatabase(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	// The filter is built from a row that is actually there, so the test does
	// not depend on what the seed happens to contain.
	var name string
	err := pool.QueryRow(ctx,
		"SELECT name FROM universities WHERE status = 'published' LIMIT 1").Scan(&name)
	if err != nil {
		t.Skipf("no published universities to search for: %v", err)
	}

	interpreter := &scriptedInterpreter{
		entity: "universities",
		dto: &FilterSetDTO{
			Entity:      "universities",
			Filters:     []FilterDTO{{Field: "name", Op: "matches", Value: name}},
			Assumptions: "read the place as the country",
		},
	}
	svc := NewService(interpreter, NewRepository(pool))

	res, err := svc.Search(ctx, SearchRequest{
		Query:    "universities in nepal",
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if res.Interpretation.Entity != "universities" {
		t.Errorf("expected the catalogue to be reported, got %q", res.Interpretation.Entity)
	}
	if res.Interpretation.Assumptions != "read the place as the country" {
		t.Errorf("expected the assumption to be reported, got %q", res.Interpretation.Assumptions)
	}
	if res.Meta.Total <= 0 {
		t.Errorf("expected some rows, got %d", res.Meta.Total)
	}
	if len(res.Items) == 0 {
		t.Fatal("expected some rows")
	}
	for _, item := range res.Items {
		if item.Title == "" {
			t.Error("expected the row to carry a title")
		}
	}
}

// A draft is out of reach however the question is phrased: the publication
// filter is added by the compiler, after the model has finished.
func TestService_SearchOnlyReturnsPublishedRows(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	// A country nothing else uses, so the only rows that can answer are the
	// two written here.
	country := "NL Search Testland"

	insert := func(status, name string) string {
		t.Helper()
		var id string
		err := pool.QueryRow(ctx,
			"INSERT INTO universities (name, slug, country, status) VALUES ($1, $2, $3, $4) RETURNING id",
			name, "nl-search-test-"+status, country, status).Scan(&id)
		if err != nil {
			t.Fatalf("insert %s: %v", status, err)
		}
		return id
	}

	publishedID := insert("published", "NL Search Test Published")
	draftID := insert("draft", "NL Search Test Draft")
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM universities WHERE id = $1", publishedID)
		pool.Exec(ctx, "DELETE FROM universities WHERE id = $1", draftID)
	})

	interpreter := &scriptedInterpreter{
		entity: "universities",
		dto: &FilterSetDTO{
			Entity:  "universities",
			Filters: []FilterDTO{{Field: "country", Op: "=", Value: country}},
		},
	}
	svc := NewService(interpreter, NewRepository(pool))

	res, err := svc.Search(ctx, SearchRequest{
		Query:    "universities in " + country,
		Page:     1,
		PageSize: 50,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	for _, item := range res.Items {
		if item.ID == draftID {
			t.Error("expected the draft row to be excluded")
		}
	}
	if res.Meta.Total != 1 {
		t.Errorf("expected only the published row, got %d", res.Meta.Total)
	}
}
