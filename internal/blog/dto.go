package blog

type CreateBlogRequest struct {
	Title             string   `json:"title"             validate:"required,min=3,max=255"`
	Slug              string   `json:"slug"              validate:"required,min=3,max=255"`
	MetaDescription   *string  `json:"meta_description"  validate:"omitempty,max=160"`
	BodyHTML          string   `json:"body_html"         validate:"required,min=1,max=500000"`
	CoverImage        *string  `json:"cover_image"       validate:"omitempty,url,max=500"`
	AuthorName        *string  `json:"author_name"       validate:"omitempty,max=255"`
	AuthorDescription *string  `json:"author_description" validate:"omitempty,max=2000"`
	AuthorTitle       *string  `json:"author_title"      validate:"omitempty,max=255"`
	Tags              []string `json:"tags"              validate:"omitempty,max=20,dive,min=1,max=50"`
	// Status is optional on create; defaults to "draft" if empty.
	Status string `json:"status" validate:"omitempty,oneof=draft published archived"`
}

type UpdateBlogRequest struct {
	Title             string   `json:"title"             validate:"required,min=3,max=255"`
	Slug              string   `json:"slug"              validate:"required,min=3,max=255"`
	MetaDescription   *string  `json:"meta_description"  validate:"omitempty,max=160"`
	BodyHTML          string   `json:"body_html"         validate:"required,min=1,max=500000"`
	CoverImage        *string  `json:"cover_image"       validate:"omitempty,url,max=500"`
	AuthorName        *string  `json:"author_name"       validate:"omitempty,max=255"`
	AuthorDescription *string  `json:"author_description" validate:"omitempty,max=2000"`
	AuthorTitle       *string  `json:"author_title"      validate:"omitempty,max=255"`
	Tags              []string `json:"tags"              validate:"omitempty,max=20,dive,min=1,max=50"`
	// Status and published_at are NOT updatable here — those move through
	// POST /blogs/{id}/publish so editors can fix typos in a live post
	// without re-publishing.
}

// PublishRequest toggles a blog between published and draft. Send
// `{"publish": true}` to publish (preserving the original published_at if
// the post was previously published) and `{"publish": false}` to unpublish
// (sets status='draft').
type PublishRequest struct {
	Publish bool `json:"publish"`
}

// BlogResponse is the full payload returned to the admin portal and to
// the public reader after a get-by-slug. Timestamps are RFC3339 strings;
// nullable fields use *string so JSON omits them entirely when nil.
type BlogResponse struct {
	ID                string   `json:"id"`
	Title             string   `json:"title"`
	Slug              string   `json:"slug"`
	MetaDescription   *string  `json:"meta_description,omitempty"`
	BodyHTML          string   `json:"body_html"`
	CoverImage        *string  `json:"cover_image,omitempty"`
	Status            string   `json:"status"`
	PublishedAt       *string  `json:"published_at,omitempty"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
	AuthorName        *string  `json:"author_name,omitempty"`
	AuthorDescription *string  `json:"author_description,omitempty"`
	AuthorTitle       *string  `json:"author_title,omitempty"`
	// Tags is always present in JSON, even when empty, so the frontend
	// can render the filter chips without a null check.
	Tags []string `json:"tags"`
}

// PublicListItem is the slim shape used by the public blog index. The full
// body_html is omitted on purpose — a 20-post list with 100 KB HTML each
// would balloon to 2 MB. The reader fetches the full post via
// GET /blogs/{slug}.
type PublicListItem struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Slug            string   `json:"slug"`
	MetaDescription *string  `json:"meta_description,omitempty"`
	CoverImage      *string  `json:"cover_image,omitempty"`
	PublishedAt     string   `json:"published_at"`
	// Tags are included on list items so the public index can render
	// filter chips without a second round-trip per post.
	Tags []string `json:"tags"`
}

// ListFilters carries query-string parameters for list endpoints.
type ListFilters struct {
	Status string
	Tags   []string
	Page   int
	Limit  int
}

// BlogListResponse is the paginated envelope used by both the admin
// /all endpoint (Items is []BlogResponse) and the public / endpoint
// (Items is []PublicListItem). The interface is loose on purpose — keeping
// a single shape avoids drift between admin and public list callers.
type BlogListResponse struct {
	Items      any   `json:"items"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}
