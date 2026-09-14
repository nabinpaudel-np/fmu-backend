package blog

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"fmu-backend/internal/db/sqlc"
)

// ErrInvalidSlug is returned by the service layer when a request supplies
// a slug that doesn't match the expected format. The handler maps this to
// 400 Bad Request.
var ErrInvalidSlug = errors.New("invalid slug: must be lowercase alphanumeric with single hyphens (e.g. my-first-post)")

// slugRe enforces the same slug format used elsewhere in the codebase
// (lowercase alphanumeric segments joined by single hyphens, no leading
// or trailing hyphens, no consecutive hyphens).
var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

const (
	maxTags       = 20
	maxTagLength  = 50
	maxAuthorName = 255
)

type BlogService interface {
	Create(ctx context.Context, req *CreateBlogRequest) (*BlogResponse, error)
	GetByID(ctx context.Context, id string) (*BlogResponse, error)
	GetPublishedBySlug(ctx context.Context, slug string) (*BlogResponse, error)
	Update(ctx context.Context, id string, req *UpdateBlogRequest) (*BlogResponse, error)
	Delete(ctx context.Context, id string) error
	Publish(ctx context.Context, id string, publish bool) (*BlogResponse, error)
	ListPublic(ctx context.Context, f ListFilters) ([]PublicListItem, int64, error)
	ListAdmin(ctx context.Context, f ListFilters) ([]BlogResponse, int64, error)
}

type blogService struct {
	repo BlogRepository
}

func NewBlogService(repo BlogRepository) BlogService {
	return &blogService{repo: repo}
}

func (s *blogService) Create(ctx context.Context, req *CreateBlogRequest) (*BlogResponse, error) {
	if err := normaliseAndValidate(req); err != nil {
		return nil, err
	}
	status := req.Status
	if status == "" {
		status = "draft"
	}
	params := sqlc.CreateBlogParams{
		Title:             req.Title,
		Slug:              req.Slug,
		MetaDescription:   req.MetaDescription,
		BodyHtml:          req.BodyHTML,
		CoverImage:        req.CoverImage,
		Status:            status,
		AuthorName:        trimPtr(req.AuthorName),
		AuthorDescription: trimPtr(req.AuthorDescription),
		AuthorTitle:       trimPtr(req.AuthorTitle),
		Tags:              req.Tags,
	}
	// If the caller asked to create-and-publish in one shot, stamp
	// published_at so the public list picks it up. PublishBlog (called via
	// the dedicated endpoint) handles this for staged publishes.
	if status == "published" {
		params.PublishedAt = pgtype.Timestamptz{Time: nowUTC(), Valid: true}
	}
	row, err := s.repo.Create(ctx, params)
	if err != nil {
		log.Default().Printf("blog create failed: %v", err)
		return nil, err
	}
	return toResponse(row), nil
}

func (s *blogService) GetByID(ctx context.Context, id string) (*BlogResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Default().Printf("blog get-by-id %s failed: %v", id, err)
		return nil, err
	}
	return toResponse(row), nil
}

// GetPublishedBySlug is the public read path: 404s for drafts, archives,
// or unknown slugs. We use the dedicated GetPublishedBlogBySlug query so
// the WHERE clause (status='published' AND published_at IS NOT NULL) is
// enforced in SQL rather than at the application layer.
func (s *blogService) GetPublishedBySlug(ctx context.Context, slug string) (*BlogResponse, error) {
	row, err := s.repo.GetPublishedBySlug(ctx, slug)
	if err != nil {
		log.Default().Printf("blog get-by-slug %s failed: %v", slug, err)
		return nil, err
	}
	return toResponse(row), nil
}

func (s *blogService) Update(ctx context.Context, id string, req *UpdateBlogRequest) (*BlogResponse, error) {
	if err := normaliseAndValidate(req); err != nil {
		return nil, err
	}
	params := sqlc.UpdateBlogParams{
		ID:                id,
		Title:             req.Title,
		Slug:              req.Slug,
		MetaDescription:   req.MetaDescription,
		BodyHtml:          req.BodyHTML,
		CoverImage:        req.CoverImage,
		AuthorName:        trimPtr(req.AuthorName),
		AuthorDescription: trimPtr(req.AuthorDescription),
		AuthorTitle:       trimPtr(req.AuthorTitle),
		Tags:              req.Tags,
	}
	row, err := s.repo.Update(ctx, params)
	if err != nil {
		log.Default().Printf("blog update %s failed: %v", id, err)
		return nil, err
	}
	return toResponse(row), nil
}

func (s *blogService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		log.Default().Printf("blog delete %s failed: %v", id, err)
		return err
	}
	return nil
}

func (s *blogService) Publish(ctx context.Context, id string, publish bool) (*BlogResponse, error) {
	var (
		row sqlc.Blog
		err error
	)
	if publish {
		row, err = s.repo.Publish(ctx, id)
	} else {
		row, err = s.repo.Unpublish(ctx, id)
	}
	if err != nil {
		log.Default().Printf("blog publish %s (publish=%v) failed: %v", id, publish, err)
		return nil, err
	}
	return toResponse(row), nil
}

func (s *blogService) ListPublic(ctx context.Context, f ListFilters) ([]PublicListItem, int64, error) {
	page, limit := clampPaging(f.Page, f.Limit)
	offset := int32((page - 1) * int(limit))
	tags := normaliseTagFilter(f.Tags)

	total, err := s.repo.CountPublished(ctx, tags)
	if err != nil {
		log.Default().Printf("blog count published failed: %v", err)
		return nil, 0, err
	}
	rows, err := s.repo.ListPublished(ctx, tags, limit, offset)
	if err != nil {
		log.Default().Printf("blog list published failed: %v", err)
		return nil, 0, err
	}
	return toPublicListItems(rows), total, nil
}

func (s *blogService) ListAdmin(ctx context.Context, f ListFilters) ([]BlogResponse, int64, error) {
	page, limit := clampPaging(f.Page, f.Limit)
	offset := int32((page - 1) * int(limit))
	tags := normaliseTagFilter(f.Tags)

	total, err := s.repo.CountAdmin(ctx, f.Status, tags)
	if err != nil {
		log.Default().Printf("blog count admin failed: %v", err)
		return nil, 0, err
	}
	rows, err := s.repo.ListAdmin(ctx, f.Status, tags, limit, offset)
	if err != nil {
		log.Default().Printf("blog list admin failed: %v", err)
		return nil, 0, err
	}
	return toResponses(rows), total, nil
}

// normaliseAndValidate trims whitespace, lowercases the slug, normalises
// tag values (trim, lowercase, dedupe), and runs the slug regex. Returns
// ErrInvalidSlug on failure. Title validation is left to validator/v10 in
// the handler.
func normaliseAndValidate(req interface{}) error {
	switch r := req.(type) {
	case *CreateBlogRequest:
		r.normaliseSlug()
		r.normaliseTitle()
		r.normaliseAuthor()
		r.Tags = cleanTags(r.Tags)
		if !slugRe.MatchString(r.Slug) {
			return ErrInvalidSlug
		}
		return nil
	case *UpdateBlogRequest:
		r.normaliseSlug()
		r.normaliseTitle()
		r.normaliseAuthor()
		r.Tags = cleanTags(r.Tags)
		if !slugRe.MatchString(r.Slug) {
			return ErrInvalidSlug
		}
		return nil
	default:
		return errors.New("normaliseAndValidate: unsupported type")
	}
}

func (r *CreateBlogRequest) normaliseSlug()    { r.Slug = strings.ToLower(strings.TrimSpace(r.Slug)) }
func (r *CreateBlogRequest) normaliseTitle()   { r.Title = strings.TrimSpace(r.Title) }
func (r *CreateBlogRequest) normaliseAuthor()  { r.AuthorName = trimPtr(r.AuthorName); r.AuthorTitle = trimPtr(r.AuthorTitle); r.AuthorDescription = trimPtr(r.AuthorDescription) }
func (r *UpdateBlogRequest) normaliseSlug()    { r.Slug = strings.ToLower(strings.TrimSpace(r.Slug)) }
func (r *UpdateBlogRequest) normaliseTitle()   { r.Title = strings.TrimSpace(r.Title) }
func (r *UpdateBlogRequest) normaliseAuthor()  { r.AuthorName = trimPtr(r.AuthorName); r.AuthorTitle = trimPtr(r.AuthorTitle); r.AuthorDescription = trimPtr(r.AuthorDescription) }

// cleanTags trims, lowercases, drops empties, and dedupes while preserving
// the first occurrence order. Validator tags cap length/count, but we
// re-enforce here so anything that bypassed the validator (e.g. direct
// service calls) still gets safe data on the way to the DB.
func cleanTags(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		t := strings.ToLower(strings.TrimSpace(raw))
		if t == "" || len(t) > maxTagLength {
			continue
		}
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
		if len(out) >= maxTags {
			break
		}
	}
	return out
}

// normaliseTagFilter lower-cases the supplied filter tags so callers don't
// have to worry about case. Returns an empty slice (not nil) so SQL sees
// `cardinality(...) = 0` and skips the tag predicate entirely.
func normaliseTagFilter(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		t := strings.ToLower(strings.TrimSpace(raw))
		if t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}

// trimPtr returns nil for pointers that point at whitespace-only strings,
// otherwise returns a pointer to the trimmed value. Keeps nullable columns
// clean of accidental "   " values that would render as blank fields.
func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*p)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// clampPaging matches the pagination package defaults: page=1, limit=20,
// limit capped at 100.
func clampPaging(page, limit int) (int, int32) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, int32(limit)
}

func nowUTC() time.Time {
	return time.Now().UTC()
}