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

	// A request that names a quality is answered as a filter as well as an
	// ordering, whether or not the model remembered to add the filter.
	applyQualities(plans, query)

	s.cache.Put(cacheKey, plans)

	sortLabel := "none"
	if plans.Primary.Sort != nil {
		sortLabel = plans.Primary.Sort.Field + " " + plans.Primary.Sort.Direction
	}
	log.Printf("nl-search: %q -> %s (search=%q, sort=%s, %d filters, %d related groups, %d catalogues)",
		query, plans.Primary.Entity.Name, plans.Primary.Search, sortLabel, len(plans.Primary.Filters),
		len(plans.Primary.Related), len(plans.All()))

	return s.run(ctx, req, query, plans, plans.Primary.Assumptions)
}

// applyQualities applies the quality a request names to every plan it spans.
// A question can be about more than one catalogue, and a quality only applies
// to the catalogues that have the field it refers to.
func applyQualities(plans *Plans, question string) {
	if plans == nil {
		return
	}
	for _, plan := range plans.All() {
		ApplyQuality(plan, question)
	}
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

	// A question can span several catalogues. Each one is searched on its
	// own and the rows are combined, so the page is taken from the combined
	// list rather than from each catalogue separately. Taking a page from
	// each would return more rows than asked for, and page two would skip
	// the rows that were not shown on page one.
	all := plans.All()

	limit := page.Limit()
	offset := page.Offset()

	// The rows are read in order and the ones the page needs are kept, so
	// the offset is applied once across every catalogue.
	items := make([]ResultItem, 0, limit)
	var total int64
	skipped := 0

	for _, plan := range all {
		rows, count, err := s.repo.Search(ctx, plan, pagination.Query{
			Page:     1,
			PageSize: maxPageSize,
		})
		if err != nil {
			return nil, err
		}
		total += count

		for _, row := range rows {
			if skipped < offset {
				skipped++
				continue
			}
			if len(items) >= limit {
				break
			}
			items = append(items, row)
		}
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
