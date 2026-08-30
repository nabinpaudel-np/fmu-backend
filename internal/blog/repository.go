package blog

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/errs"
)

type BlogRepository interface {
	Create(ctx context.Context, arg sqlc.CreateBlogParams) (sqlc.Blog, error)
	GetByID(ctx context.Context, id string) (sqlc.Blog, error)
	GetBySlug(ctx context.Context, slug string) (sqlc.Blog, error)
	GetPublishedBySlug(ctx context.Context, slug string) (sqlc.Blog, error)
	Update(ctx context.Context, arg sqlc.UpdateBlogParams) (sqlc.Blog, error)
	Delete(ctx context.Context, id string) error
	Publish(ctx context.Context, id string) (sqlc.Blog, error)
	Unpublish(ctx context.Context, id string) (sqlc.Blog, error)
	ListPublished(ctx context.Context, tags []string, limit, offset int32) ([]sqlc.Blog, error)
	CountPublished(ctx context.Context, tags []string) (int64, error)
	ListAdmin(ctx context.Context, status string, tags []string, limit, offset int32) ([]sqlc.Blog, error)
	CountAdmin(ctx context.Context, status string, tags []string) (int64, error)
}

type blogRepository struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
}

func NewBlogRepository(queries *sqlc.Queries, pool *pgxpool.Pool) BlogRepository {
	return &blogRepository{queries: queries, pool: pool}
}

func (r *blogRepository) Create(ctx context.Context, arg sqlc.CreateBlogParams) (sqlc.Blog, error) {
	row, err := r.queries.CreateBlog(ctx, arg)
	if err != nil {
		return sqlc.Blog{}, translateErr(err, fmt.Sprintf("slug=%s", arg.Slug))
	}
	return row, nil
}

func (r *blogRepository) GetByID(ctx context.Context, id string) (sqlc.Blog, error) {
	row, err := r.queries.GetBlogByID(ctx, id)
	if err != nil {
		return sqlc.Blog{}, translateErr(err, fmt.Sprintf("id=%s", id))
	}
	return row, nil
}

func (r *blogRepository) GetBySlug(ctx context.Context, slug string) (sqlc.Blog, error) {
	row, err := r.queries.GetBlogBySlug(ctx, slug)
	if err != nil {
		return sqlc.Blog{}, translateErr(err, fmt.Sprintf("slug=%s", slug))
	}
	return row, nil
}

func (r *blogRepository) GetPublishedBySlug(ctx context.Context, slug string) (sqlc.Blog, error) {
	row, err := r.queries.GetPublishedBlogBySlug(ctx, slug)
	if err != nil {
		return sqlc.Blog{}, translateErr(err, fmt.Sprintf("slug=%s", slug))
	}
	return row, nil
}

func (r *blogRepository) Update(ctx context.Context, arg sqlc.UpdateBlogParams) (sqlc.Blog, error) {
	row, err := r.queries.UpdateBlog(ctx, arg)
	if err != nil {
		return sqlc.Blog{}, translateErr(err, fmt.Sprintf("id=%s slug=%s", arg.ID, arg.Slug))
	}
	return row, nil
}

func (r *blogRepository) Delete(ctx context.Context, id string) error {
	rows, err := r.queries.DeleteBlog(ctx, id)
	if err != nil {
		return translateErr(err, fmt.Sprintf("id=%s", id))
	}
	if rows == 0 {
		return errs.ErrNotFound
	}
	return nil
}

func (r *blogRepository) Publish(ctx context.Context, id string) (sqlc.Blog, error) {
	row, err := r.queries.PublishBlog(ctx, id)
	if err != nil {
		return sqlc.Blog{}, translateErr(err, fmt.Sprintf("id=%s", id))
	}
	return row, nil
}

func (r *blogRepository) Unpublish(ctx context.Context, id string) (sqlc.Blog, error) {
	row, err := r.queries.UnpublishBlog(ctx, id)
	if err != nil {
		return sqlc.Blog{}, translateErr(err, fmt.Sprintf("id=%s", id))
	}
	return row, nil
}

func (r *blogRepository) ListPublished(ctx context.Context, tags []string, limit, offset int32) ([]sqlc.Blog, error) {
	if tags == nil {
		tags = []string{}
	}
	rows, err := r.queries.ListPublishedBlogs(ctx, sqlc.ListPublishedBlogsParams{
		Column1: tags,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list published blogs: %w", err)
	}
	return rows, nil
}

func (r *blogRepository) CountPublished(ctx context.Context, tags []string) (int64, error) {
	if tags == nil {
		tags = []string{}
	}
	n, err := r.queries.CountPublishedBlogs(ctx, tags)
	if err != nil {
		return 0, fmt.Errorf("count published blogs: %w", err)
	}
	return n, nil
}

func (r *blogRepository) ListAdmin(ctx context.Context, status string, tags []string, limit, offset int32) ([]sqlc.Blog, error) {
	if tags == nil {
		tags = []string{}
	}
	rows, err := r.queries.ListBlogsAdmin(ctx, sqlc.ListBlogsAdminParams{
		Column1: status,
		Column2: tags,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list blogs admin: %w", err)
	}
	return rows, nil
}

func (r *blogRepository) CountAdmin(ctx context.Context, status string, tags []string) (int64, error) {
	if tags == nil {
		tags = []string{}
	}
	n, err := r.queries.CountBlogsAdmin(ctx, sqlc.CountBlogsAdminParams{
		Column1: status,
		Column2: tags,
	})
	if err != nil {
		return 0, fmt.Errorf("count blogs admin: %w", err)
	}
	return n, nil
}

// translateErr centralises PG error → sentinel mapping. The context
// argument is appended to wrapped errors so admin debug logs can identify
// the offending row without searching through structured logs.
func translateErr(err error, context string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return errs.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			// Unique violation on slug — differentiate from any other
			// future unique constraint so callers can return 409 with a
			// meaningful message.
			if pgErr.ConstraintName == "idx_blogs_slug" || pgErr.ConstraintName == "blogs_slug_key" {
				return fmt.Errorf("%w (%s)", errs.ErrBlogSlugTaken, context)
			}
			return fmt.Errorf("%w (%s)", errs.ErrBadRequest, context)
		case "22P02":
			// invalid_text_representation — typically a malformed UUID.
			return errs.ErrNotFound
		}
	}
	return err
}