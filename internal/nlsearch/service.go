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
// back, and the plan is executed against the catalogue. It is the only place
// that knows the order of those steps.

// maxPageSize caps how many rows one request may ask for. The model call is
// the expensive part, not the query, but an unbounded page size is still a
// way to pull the whole catalogue.
const maxPageSize = 50

// maxQueryLength caps the question. The model is paid per token, so an
// unbounded question is an unbounded bill, and a question that long is not a
// question anyone is typing.
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
}

// NewService builds the service. A nil interpreter is allowed: the endpoint
// then reports that the feature is not configured rather than failing at
// boot.
func NewService(interpreter Interpreter, repo Repository) Service {
	return &service{interpreter: interpreter, repo: repo}
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

	// The catalogue is decided first, so the model is asked about fields
	// that belong to it and cannot name one that does not.
	classification, err := s.interpreter.Classify(ctx, query)
	if err != nil {
		return nil, err
	}

	dto, err := s.interpreter.Interpret(ctx, InterpretRequest{
		Question: query,
		Entity:   classification.Entity,
	})
	if err != nil {
		return nil, err
	}

	plan, verr := Validate(dto)
	if verr != nil {
		// The model is given the exact complaint once. A second failure is
		// reported to the caller rather than retried again.
		dto, err = s.interpreter.Interpret(ctx, InterpretRequest{
			Question:   query,
			Entity:     classification.Entity,
			Correction: verr.Error(),
		})
		if err != nil {
			return nil, err
		}
		plan, verr = Validate(dto)
		if verr != nil {
			return nil, verr
		}
	}

	log.Printf("nl-search: %q -> %s (%d filters, %d related groups)",
		query, plan.Entity.Name, len(plan.Filters), len(plan.Related))

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

	items, total, err := s.repo.Search(ctx, plan, page)
	if err != nil {
		return nil, err
	}

	return &SearchResponse{
		Query: query,
		Interpretation: Interpretation{
			Entity:      plan.Entity.Name,
			Assumptions: joinNotes(classification.Assumptions, plan.Assumptions),
			Unmapped:    plan.Unmapped,
		},
		Items: items,
		Meta:  page.BuildMeta(total),
	}, nil
}

// joinNotes merges the two notes the model may have left, one from deciding
// the catalogue and one from reading the question.
func joinNotes(notes ...string) string {
	kept := make([]string, 0, len(notes))
	for _, note := range notes {
		if note = strings.TrimSpace(note); note != "" {
			kept = append(kept, note)
		}
	}
	return strings.Join(kept, "; ")
}
