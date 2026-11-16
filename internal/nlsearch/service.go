package nlsearch

import (
	"context"
	"errors"
	"log"
	"strings"
	"unicode/utf8"

	"fmu-backend/internal/pagination"
)

// The service is the whole flow: a question goes in, an interpretation comes
// back, and the plan is executed against the catalogue.

// maxPageSize caps how many rows one request may ask for.
const maxPageSize = 50

// maxQueryLength caps the question. The model is paid per token, so an
// unbounded question is an unbounded bill.
const maxQueryLength = 500

// ErrEmptyQuery means the request carried no question.
var ErrEmptyQuery = errors.New("a query is required")

// ErrQueryTooLong means the question was longer than the cap.
var ErrQueryTooLong = errors.New("the query is too long")

// Service answers natural-language questions about the catalogue.
type Service interface {
	Search(ctx context.Context, req SearchRequest) (*SearchResponse, error)
}

type service struct {
	interpreter Interpreter
	repo        Repository
	cache       *planCache
}

// NewService builds the service. A nil interpreter is allowed.
func NewService(interpreter Interpreter, repo Repository) Service {
	return &service{interpreter: interpreter, repo: repo, cache: newPlanCache()}
}

// Search interprets the question and runs the resulting plan.
func (s *service) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil, ErrEmptyQuery
	}
	if utf8.RuneCountInString(query) > maxQueryLength {
		return nil, ErrQueryTooLong
	}
	if s.interpreter == nil {
		return nil, ErrProviderUnavailable
	}

	// The model reads the question and decides which catalogue it is about,
	// so there is nothing decided here. The server checks afterwards that
	// what the model named exists.
	cacheKey := normaliseQuery(query)
	if cached, ok := s.cache.Get(cacheKey); ok {
		log.Printf("nl-search: %q -> %s (cached)", query, cached.Primary.Entity.Name)
		return s.run(ctx, req, query, cached, cached.Primary.Assumptions)
	}

	dto, err := s.interpreter.Interpret(ctx, InterpretRequest{Question: query})
	if err != nil {
		return nil, err
	}

	plans, verr := Validate(dto)
	if verr != nil {
		// The model is given the exact complaint once. A second failure is
		// reported to the caller rather than retried again.
		dto, err = s.interpreter.Interpret(ctx, InterpretRequest{
			Question:   query,
			Correction: verr.Error(),
		})
		if err != nil {
			return nil, err
		}
		plans, verr = Validate(dto)
		if verr != nil {
			return nil, verr
		}
	}

	s.cache.Put(cacheKey, plans)

	log.Printf("nl-search: %q -> %s (%d filters, %d related groups, %d catalogues)",
		query, plans.Primary.Entity.Name, len(plans.Primary.Filters),
		len(plans.Primary.Related), len(plans.All()))

	return s.run(ctx, req, query, plans, plans.Primary.Assumptions)
}

// run executes the plans against the catalogue and shapes the answer.
func (s *service) run(ctx context.Context, req SearchRequest, query string, plans *Plans, assumptions string) (*SearchResponse, error) {
	page := pagination.Query{Page: req.Page, PageSize: req.PageSize}
	if page.Page < 1 {
		page.Page = pagination.DefaultPage
	}
	if page.PageSize < 1 {
		page.PageSize = pagination.DefaultPageSize
	}
	if page.PageSize > maxPageSize {
		page.PageSize = maxPageSize
	}

	items := make([]ResultItem, 0, page.PageSize)
	var total int64

	for _, plan := range plans.All() {
		rows, count, err := s.repo.Search(ctx, plan, page)
		if err != nil {
			return nil, err
		}
		items = append(items, rows...)
		total += count
	}

	return &SearchResponse{
		Query: query,
		Interpretation: Interpretation{
			Entity:      strings.Join(plans.Entities(), ", "),
			Assumptions: assumptions,
			Unmapped:    plans.Primary.Unmapped,
		},
		Items: items,
		Meta:  page.BuildMeta(total),
	}, nil
}
