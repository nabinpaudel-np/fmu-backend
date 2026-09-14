-- name: CreateBlog :one
INSERT INTO blogs (
    title, slug, meta_description, body_html, cover_image, status, published_at,
    author_name, author_description, author_title, tags
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11
)
RETURNING id, title, slug, meta_description, body_html, cover_image,
          status, published_at, created_at, updated_at,
          author_name, author_description, author_title, tags;

-- name: GetBlogByID :one
SELECT id, title, slug, meta_description, body_html, cover_image,
       status, published_at, created_at, updated_at,
       author_name, author_description, author_title, tags
FROM blogs
WHERE id = $1;

-- name: GetBlogBySlug :one
SELECT id, title, slug, meta_description, body_html, cover_image,
       status, published_at, created_at, updated_at,
       author_name, author_description, author_title, tags
FROM blogs
WHERE slug = $1;

-- name: GetPublishedBlogBySlug :one
SELECT id, title, slug, meta_description, body_html, cover_image,
       status, published_at, created_at, updated_at,
       author_name, author_description, author_title, tags
FROM blogs
WHERE slug = $1
  AND status = 'published'
  AND published_at IS NOT NULL;

-- name: UpdateBlog :one
UPDATE blogs
SET title = $2,
    slug = $3,
    meta_description = $4,
    body_html = $5,
    cover_image = $6,
    author_name = $7,
    author_description = $8,
    author_title = $9,
    tags = $10,
    updated_at = NOW()
WHERE id = $1
RETURNING id, title, slug, meta_description, body_html, cover_image,
          status, published_at, created_at, updated_at,
          author_name, author_description, author_title, tags;

-- name: DeleteBlog :execrows
DELETE FROM blogs WHERE id = $1;

-- name: PublishBlog :one
-- Stamps published_at only on the first draft -> published transition.
-- Re-publishing a previously published blog preserves the original timestamp.
UPDATE blogs
SET status = 'published',
    published_at = COALESCE(published_at, NOW()),
    updated_at = NOW()
WHERE id = $1
RETURNING id, title, slug, meta_description, body_html, cover_image,
          status, published_at, created_at, updated_at,
          author_name, author_description, author_title, tags;

-- name: UnpublishBlog :one
UPDATE blogs
SET status = 'draft',
    updated_at = NOW()
WHERE id = $1
RETURNING id, title, slug, meta_description, body_html, cover_image,
          status, published_at, created_at, updated_at,
          author_name, author_description, author_title, tags;

-- name: ListPublishedBlogs :many
-- filter_tags is a TEXT[] of tags to filter by. NULL or '{}' means no
-- tag filter. Uses the && (overlap) operator — OR semantics: a post is
-- returned if it has ANY of the requested tags. Indexed via idx_blogs_tags.
SELECT id, title, slug, meta_description, body_html, cover_image,
       status, published_at, created_at, updated_at,
       author_name, author_description, author_title, tags
FROM blogs
WHERE status = 'published'
  AND published_at IS NOT NULL
  AND (cardinality($1::text[]) = 0 OR tags && $1::text[])
ORDER BY published_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPublishedBlogs :one
SELECT COUNT(*)::bigint FROM blogs
WHERE status = 'published'
  AND published_at IS NOT NULL
  AND (cardinality($1::text[]) = 0 OR tags && $1::text[]);

-- name: ListBlogsAdmin :many
SELECT id, title, slug, meta_description, body_html, cover_image,
       status, published_at, created_at, updated_at,
       author_name, author_description, author_title, tags
FROM blogs
WHERE (NULLIF($1, '') IS NULL OR status = $1)
  AND (cardinality($2::text[]) = 0 OR tags && $2::text[])
ORDER BY COALESCE(published_at, created_at) DESC
LIMIT $3 OFFSET $4;

-- name: CountBlogsAdmin :one
SELECT COUNT(*)::bigint FROM blogs
WHERE (NULLIF($1, '') IS NULL OR status = $1)
  AND (cardinality($2::text[]) = 0 OR tags && $2::text[]);
