# Blog CMS — Frontend Integration Guide

Backend implementation lives on branch `feat/blog`. This document covers everything the frontend needs to integrate the Admin Portal blog editor and the public-facing blog index/article templates.

## Table of Contents

1. [Overview](#overview)
2. [Auth & Cookies](#auth--cookies)
3. [Quick Reference: All Endpoints](#quick-reference-all-endpoints)
4. [Image Upload (Cover + Inline)](#image-upload-cover--inline)
5. [Endpoints in Detail](#endpoints-in-detail)
   - [Admin: Create Blog](#admin-create-blog)
   - [Admin: Update Blog](#admin-update-blog)
   - [Admin: Delete Blog](#admin-delete-blog)
   - [Admin: Publish / Unpublish](#admin-publish--unpublish)
   - [Admin: List All (any status)](#admin-list-all-any-status)
   - [Admin: Get By ID](#admin-get-by-id)
   - [Public: List Published](#public-list-published)
   - [Public: Get By Slug](#public-get-by-slug)
6. [Author & Tags](#author--tags)
7. [Tag Filtering](#tag-filtering)
8. [Request & Response Shapes](#request--response-shapes)
9. [Validation Rules](#validation-rules)
10. [Error Responses](#error-responses)
11. [Pagination](#pagination)
12. [WYSIWYG Editor: HTML Payload & Sanitisation](#wysiwyg-editor-html-payload--sanitisation)
13. [Curl Examples (End-to-End)](#curl-examples-end-to-end)

---

## Overview

The blog resource is a single table (`public.blogs`) that backs two consumers:

| Consumer | What it shows | Auth |
|----------|---------------|------|
| Public blog index + article templates | Only `status='published'` posts, metadata-only on lists | None |
| Admin Portal "Blogs" page | All posts (draft, published, archived) with full body | Admin JWT cookie |

State machine:

```
draft  ──publish──►  published  ──unpublish──►  draft
   │                       │
   │                       └──► published (idempotent re-publish keeps original published_at)
   │
   └──► (delete)            ──► deleted (hard delete, no soft delete in this codebase)
```

`archived` is a valid DB status (kept in the CHECK constraint for forward-compat) but **v1 has no API to set it**. Use `DELETE` to remove a blog.

### Important Backend Decisions

- **Slug is required from the client** — the server does NOT auto-generate it from the title.
- **`published_at` is stamped once** — on the first `draft → published` transition. Subsequent re-publishes preserve the original timestamp so the public sort order and any future RSS/SEO signals stay stable.
- **Updating a published blog does NOT touch `status` or `published_at`** — only the content fields. Editors can fix typos without "re-publishing".
- **Public list omits `body_html`** — keeps list payloads small. Get the full post via the by-slug endpoint.
- **Tags are normalised server-side** — trimmed, lowercased, and de-duplicated (preserving first-seen order). The response always reflects the canonical stored form. Filter tags in the query string are normalised the same way, so callers don't have to worry about case.
- **Backend does NOT sanitise HTML** — see [WYSIWYG section](#wysiwyg-editor-html-payload--sanitisation).

---

## Auth & Cookies

All admin endpoints require an **HttpOnly cookie** named `access_token`. The login flow sets it:

```
POST /api/v1/auth/login
{ "email": "...", "password": "..." }

→ Set-Cookie: access_token=eyJ...; HttpOnly; Path=/; Max-Age=1800
```

The frontend should send `credentials: 'include'` (fetch) or `withCredentials: true` (axios) on every admin request. The browser will attach the cookie automatically.

**Public endpoints (`GET /api/v1/blogs`, `GET /api/v1/blogs/{slug}`) require no auth.**

**Refreshing tokens:** when an admin call returns `401`, call `POST /api/v1/auth/refresh` (also cookie-based) to mint a new access token, then retry. The existing auth interceptor on the Admin Portal should already handle this.

---

## Quick Reference: All Endpoints

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| `GET`    | `/api/v1/blogs`                  | none          | Public list of published posts (metadata only). Supports `?tags=...` filter. |
| `GET`    | `/api/v1/blogs?tags=foo,bar`     | none          | Public list filtered by tags (OR semantics) |
| `GET`    | `/api/v1/blogs/{slug}`           | none          | Public single post (full body, includes author + tags) — 404 for drafts |
| `GET`    | `/api/v1/blogs/all`              | admin         | Admin list of all posts (any status) — full body |
| `GET`    | `/api/v1/blogs/all?status=draft&tags=...` | admin | Filter admin list by status and/or tags |
| `GET`    | `/api/v1/blogs/id/{id}`          | admin         | Admin single post by UUID (any status) |
| `POST`   | `/api/v1/blogs`                  | admin         | Create a blog (with author + tags) |
| `PUT`    | `/api/v1/blogs/{id}`             | admin         | Update a blog (content + author + tags; `status` unchanged) |
| `PATCH`  | `/api/v1/blogs/{id}`             | admin         | Alias for PUT |
| `DELETE` | `/api/v1/blogs/{id}`             | admin         | Hard-delete a blog |
| `POST`   | `/api/v1/blogs/{id}/publish`     | admin         | Publish (body `{"publish": true}`) or unpublish (`false`) |
| `POST`   | `/api/v1/uploads/image?purpose=blog` | admin     | Upload a cover or inline image to Cloudinary |

---

## Image Upload (Cover + Inline)

Both cover images and inline WYSIWYG images use the **same endpoint** with `purpose=blog`. The Cloudinary folder becomes `fmu/<env>/blog/...` automatically.

### Endpoint

```
POST /api/v1/uploads/image?purpose=blog
Content-Type: multipart/form-data; boundary=...
Cookie: access_token=...

--boundary
Content-Disposition: form-data; name="file"; filename="cover.jpg"
Content-Type: image/jpeg

<binary>
--boundary--
```

**Allowed MIME types:** `image/jpeg`, `image/png`, `image/webp`, `image/gif`
**Max size:** 10 MiB
**Auth:** admin role required (students get 403)

### Response

```json
{
  "success": true,
  "data": {
    "secure_url": "https://res.cloudinary.com/r6tjulal/image/upload/v1788113675/fmu/development/blog/abc123.png",
    "url":        "http://res.cloudinary.com/r6tjulal/image/upload/v1788113675/fmu/development/blog/abc123.png",
    "public_id":  "fmu/development/blog/abc123",
    "width":      1920,
    "height":     1080,
    "format":     "png",
    "bytes":      245678
  }
}
```

Use `data.secure_url` for the value stored in `cover_image` or as the `src` of an `<img>` inserted into the editor.

### Inline Image Flow

When the user clicks "insert image" in the WYSIWYG editor:

1. Open a file picker.
2. POST the chosen file to `/api/v1/uploads/image?purpose=blog` with `credentials: 'include'`.
3. Take the returned `secure_url`.
4. Insert `<img src="<secure_url>" alt="..." />` at the cursor in the editor.

---

## Endpoints in Detail

### Admin: Create Blog

```
POST /api/v1/blogs
Cookie: access_token=...
Content-Type: application/json

{
  "title":             "How to choose a university",
  "slug":              "how-to-choose-a-university",
  "meta_description":  "A short SEO blurb shown in search results",
  "body_html":         "<p>This is the <strong>article body</strong>.</p><img src=\"https://res.cloudinary.com/.../inline.png\" />",
  "cover_image":       "https://res.cloudinary.com/.../cover.png",
  "author_name":       "Jane Doe",
  "author_description":"Senior admissions counselor with 10 years of experience helping students find the right fit.",
  "author_title":      "Admissions Counselor",
  "tags":              ["Admissions", "Campus Life", "Freshman"],
  "status":            "draft"
}
```

| Field | Required | Notes |
|-------|----------|-------|
| `title` | yes | 3–255 chars (trimmed) |
| `slug` | yes | 3–255 chars, must match `^[a-z0-9]+(?:-[a-z0-9]+)*$` — lowercase, digits, single hyphens, no leading/trailing hyphens |
| `meta_description` | no | up to 160 chars (SEO) |
| `body_html` | yes | 1–500,000 chars (not sanitised server-side) |
| `cover_image` | no | URL, up to 500 chars |
| `author_name` | no | up to 255 chars. Whitespace-only is stored as `null`. |
| `author_description` | no | up to 2000 chars (long bio). Whitespace-only is stored as `null`. |
| `author_title` | no | up to 255 chars (e.g. "Admissions Counselor"). Whitespace-only is stored as `null`. |
| `tags` | no | array of strings, max 20 entries, each 1–50 chars. Normalised server-side (trimmed, lowercased, de-duplicated). Whitespace-only entries are dropped. |
| `status` | no | one of `draft`/`published`/`archived`; defaults to `draft` |

**Response `201 Created`:** full `BlogResponse` (see [shapes](#request--response-shapes)).

### Admin: Update Blog

```
PUT /api/v1/blogs/{id}
```

Same body as Create, minus `status`. Touches only content fields + author + tags — **`status` and `published_at` are NOT changed**. To flip status, use `POST /{id}/publish`.

Sending `tags: []` on update clears all tags (since an empty array round-trips as `[]`, not `null`).

**Response `200 OK`:** updated `BlogResponse`.

### Admin: Delete Blog

```
DELETE /api/v1/blogs/{id}
```

**Response `204 No Content`** on success, `404 Not Found` if the ID doesn't exist.

### Admin: Publish / Unpublish

```
POST /api/v1/blogs/{id}/publish
{ "publish": true }   # publish (or re-publish; preserves original published_at)
{ "publish": false }  # unpublish (status='draft')
```

**Response `200 OK`:** updated `BlogResponse` with `status` and `published_at` reflecting the new state.

### Admin: List All (any status)

```
GET /api/v1/blogs/all?status=draft&tags=admissions&page=1&page_size=20
```

Query params (all optional):

| Param | Values | Default | Notes |
|-------|--------|---------|-------|
| `status` | `draft`/`published`/`archived`/empty | empty (all) | empty string returns all |
| `tags` | string or repeated query | none | Filter by one or more tags (OR semantics). Accepts `?tags=a,b` (comma-separated) or `?tags=a&tags=b` (repeated). Case-insensitive. |
| `page` | int | 1 | 1-indexed |
| `page_size` (or `limit`) | int | 20 | max 100 |

**Response `200 OK`:**

```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id":               "e9bba497-b14d-4ce9-95d9-f39695f16b8e",
        "title":            "How to choose a university",
        "slug":             "how-to-choose-a-university",
        "meta_description": "A short SEO blurb shown in search results",
        "body_html":        "<p>...</p>",
        "cover_image":      "https://...",
        "status":           "published",
        "published_at":     "2026-08-30T18:14:10Z",
        "created_at":       "2026-08-30T18:14:10Z",
        "updated_at":       "2026-08-30T18:15:07Z",
        "author_name":      "Jane Doe",
        "author_description":"Senior admissions counselor with 10 years...",
        "author_title":     "Admissions Counselor",
        "tags":             ["admissions", "campus life", "freshman"]
      }
    ],
    "page": 1,
    "limit": 20,
    "total": 42,
    "total_pages": 3
  }
}
```

**Admin list items include `body_html`.** Each item is a full `BlogResponse` (including author + tags).

### Admin: Get By ID

```
GET /api/v1/blogs/id/{id}
```

UUID required. Returns the full `BlogResponse` regardless of status (use this in the editor for drafts).

### Public: List Published

```
GET /api/v1/blogs?tags=admissions,campus%20life&page=1&page_size=20
```

No auth. Returns only `status='published'` posts, ordered by `published_at DESC`. Supports the same `tags` filter as the admin list — see [Tag Filtering](#tag-filtering).

**Response shape — same envelope, but each item is a `PublicListItem`:**

```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id":               "e9bba497-b14d-4ce9-95d9-f39695f16b8e",
        "title":            "How to choose a university",
        "slug":             "how-to-choose-a-university",
        "meta_description": "A short SEO blurb shown in search results",
        "cover_image":      "https://...",
        "published_at":     "2026-08-30T18:14:10Z",
        "tags":             ["admissions", "campus life", "freshman"]
      }
    ],
    "page": 1,
    "limit": 20,
    "total": 12,
    "total_pages": 1
  }
}
```

**`PublicListItem` deliberately omits `body_html`** — fetch the full post via `GET /blogs/{slug}`. Tags are included so the public index can render filter chips without a second round-trip per post.

### Public: Get By Slug

```
GET /api/v1/blogs/{slug}
```

No auth. Returns the full `BlogResponse` (including author + tags) if the post is `published`. **Returns `404`** if the slug is unknown or the post is a draft/archived (so drafts stay private until published).

---

## Author & Tags

### Author fields

Each blog can carry an optional byline:

| Field | Max length | When to show |
|-------|------------|--------------|
| `author_name` | 255 | Bold name byline on the article template, e.g. "By Jane Doe" |
| `author_title` | 255 | Lighter line under the name, e.g. "Admissions Counselor" |
| `author_description` | 2000 | Hover-card or "About the author" block at the end of the article |

All three are `*string` — omitted from JSON when `null`. Sending a whitespace-only string is normalised to `null` server-side, so the editor can leave them blank without producing empty fields in the response.

### Tags

Tags are a flat `string[]` of short, lowercase labels. They're intended for **filter chips** on the public index and admin lists — not a deep taxonomy.

**Behaviour:**

- Server normalises: trims surrounding whitespace, lowercases, drops empties, de-duplicates (first-seen wins), caps at 20 entries, caps each entry at 50 chars.
- Storage: a PostgreSQL `TEXT[]` column with a GIN index (`idx_blogs_tags`) for fast overlap queries.
- Filtering: OR semantics (`tags && $1::text[]`) — a post matches if it has **any** of the requested tags. See [Tag Filtering](#tag-filtering).
- Response shape: tags always render as a JSON array (never `null`) so the frontend can render `tags.map(...)` without a null check.

**Example normalisation:**

```
Input:  ["Admissions", "  admissions ", "Campus Life", "", "admissions"]
Stored: ["admissions", "campus life"]
```

**Recommended UI conventions:**

- Display: show tags as lowercase chips by default — they're already stored that way.
- Editor input: a tag-input component that lets the user type and press Enter / comma to add. Lower-case the display string client-side if your UI is case-sensitive (e.g. capitalises the first letter for display but the submitted value should be lowercase).
- Cap at ~20 tags and ~50 chars per tag client-side too, so the editor gives feedback before the server rejects the request.

---

## Tag Filtering

Both the public list (`GET /api/v1/blogs`) and the admin list (`GET /api/v1/blogs/all`) accept a `tags` query parameter.

### Accepted shapes

The server accepts either of these — pick whichever is easier for the client:

```http
GET /api/v1/blogs?tags=admissions,campus%20life
GET /api/v1/blogs?tags=admissions&tags=campus%20life
```

Empty entries (e.g. `?tags=,foo,`) are dropped. Filter tags are case-insensitive — `?tags=Admissions` matches stored `"admissions"`.

### Semantics

**OR semantics.** A post is returned if it carries **any** of the requested tags. Implementation: PostgreSQL `&&` (overlap) operator on the `TEXT[]` column, backed by the GIN index.

```
GET /api/v1/blogs?tags=admissions
→ posts tagged "admissions" OR "admissions,scholarships" OR "admissions,campus life"

GET /api/v1/blogs?tags=admissions,scholarships
→ posts tagged with admissions OR scholarships (or both)
```

If you need AND semantics later ("posts tagged with BOTH admissions and scholarships"), that's a future endpoint — v1 only ships OR.

### Combining with other filters

`tags` composes with `status` on the admin list. They're AND'd together (a post must match both):

```http
GET /api/v1/blogs/all?status=draft&tags=admissions
→ admin-visible posts in draft status that are tagged with "admissions"
```

### Empty filter

Sending `?tags=` (empty), or omitting the param entirely, applies **no** tag filter. The list returns every post that matches the other criteria (status for admin, published-only for public).

---

## Request & Response Shapes

### `BlogResponse` (full payload)

```json
{
  "id":               "e9bba497-b14d-4ce9-95d9-f39695f16b8e",
  "title":            "How to choose a university",
  "slug":             "how-to-choose-a-university",
  "meta_description": "A short SEO blurb shown in search results",
  "body_html":        "<p>HTML body produced by the WYSIWYG editor.</p>",
  "cover_image":      "https://res.cloudinary.com/.../cover.png",
  "status":           "published",
  "published_at":     "2026-08-30T18:14:10Z",
  "created_at":       "2026-08-30T18:14:10Z",
  "updated_at":       "2026-08-30T18:15:07Z",
  "author_name":      "Jane Doe",
  "author_description":"Senior admissions counselor with 10 years of experience.",
  "author_title":     "Admissions Counselor",
  "tags":             ["admissions", "campus life", "freshman"]
}
```

- All timestamps are RFC3339 UTC strings.
- `meta_description`, `cover_image`, `author_name`, `author_description`, `author_title` are **omitted from JSON when null** (`omitempty`).
- `published_at` is omitted when the post has never been published.
- `tags` is **always present**, rendered as `[]` when empty (not `null`).

### `PublicListItem` (slim payload)

```json
{
  "id":               "e9bba497-b14d-4ce9-95d9-f39695f16b8e",
  "title":            "How to choose a university",
  "slug":             "how-to-choose-a-university",
  "meta_description": "A short SEO blurb shown in search results",
  "cover_image":      "https://res.cloudinary.com/.../cover.png",
  "published_at":     "2026-08-30T18:14:10Z",
  "tags":             ["admissions", "campus life", "freshman"]
}
```

No `body_html`, no `status`, no `updated_at`, no `created_at`, no author fields — those are only on the full payload. Tags ARE included on list items so the public index can render filter chips without an extra round-trip per post.

### Error Envelope

All errors use the same shape (already used everywhere else in this API):

```json
{ "success": false, "error": "blog with this slug already exists (slug=how-to-choose-a-university)" }
```

For field-level validation errors:

```json
{ "success": false, "errors": [ { "field": "title", "message": "title is required" } ] }
```

If the editor sends an empty tag string (e.g. `tags: ["a", "", "b"]`), the validator returns:

```json
{ "success": false, "errors": [ { "field": "tags[1]", "message": "tags[1] must be at least 1 characters" } ] }
```

The server never silently drops empty tags — it rejects the request. Strip empties client-side before submitting to avoid this.

---

## Validation Rules

Server-side validator (using `go-playground/validator`):

| Field | Rule |
|-------|------|
| `title` | `required,min=3,max=255` (trimmed server-side) |
| `slug` | `required,min=3,max=255` + server-side regex `^[a-z0-9]+(?:-[a-z0-9]+)*$` |
| `meta_description` | `omitempty,max=160` |
| `body_html` | `required,min=1,max=500000` |
| `cover_image` | `omitempty,url,max=500` |
| `author_name` | `omitempty,max=255` |
| `author_description` | `omitempty,max=2000` |
| `author_title` | `omitempty,max=255` |
| `tags` | `omitempty,dive,min=1,max=50,max=20` (each tag 1–50 chars, max 20 tags) |
| `status` (create only) | `omitempty,oneof=draft published archived` |

The slug is also normalised server-side (trimmed, lowercased) before validation. The frontend may display the user's input verbatim but the stored value will be the normalised one — **use the slug returned in the response for any subsequent calls, not the value the user typed.**

Tags are normalised in the service layer (trim → lowercase → drop empties → dedupe → cap at 20) on top of the validator rules. The validator rejects empty entries (`min=1`), so the editor should strip empties before submitting.

---

## Error Responses

| Status | When | Body |
|--------|------|------|
| `400 Bad Request` | Invalid JSON, validation error, invalid slug | `{ "error": "..." }` or `{ "errors": [{field,message}] }` |
| `401 Unauthorized` | Missing or invalid access_token cookie on an admin route | `{ "error": "unauthorized" }` |
| `403 Forbidden` | Authenticated but not admin (e.g. student tries to create a blog or upload to `purpose=blog`) | `{ "error": "..." }` |
| `404 Not Found` | Blog ID/slug not found, OR draft/archived post requested via public endpoint | `{ "error": "blog not found" }` |
| `409 Conflict` | Slug already exists | `{ "error": "blog with this slug already exists (slug=...)" }` |
| `413 Payload Too Large` | Image upload > 10 MiB | `{ "error": "file exceeds 10485760 bytes" }` |
| `415 Unsupported Media Type` | Image upload with disallowed MIME type | `{ "error": "mime type ... is not allowed" }` |
| `500 Internal Server Error` | Anything unexpected | `{ "error": "something went wrong" }` |

---

## Pagination

Both list endpoints support the same query parameters:

| Param | Type | Default | Max |
|-------|------|---------|-----|
| `page` | int (1-indexed) | `1` | — |
| `page_size` | int | `20` | `100` |
| `limit` | int (alias for `page_size`) | `20` | `100` |

The response envelope always includes `page`, `limit`, `total`, and `total_pages`. Compute UI pagination from `total_pages`.

`page`/`page_size` < 1 or > max are silently clamped.

When a tag filter is applied, `total` reflects the filtered count (not the total number of published posts). UI pagination should be derived from this filtered total.

---

## WYSIWYG Editor: HTML Payload & Sanitisation

### Format

`body_html` is a raw HTML string. The WYSIWYG editor is responsible for producing it; the server treats it as opaque text.

### Required features (from `feature.md`)

> bolding, italics, underlining, hyperlinks, inline image insertion

A standard editor (TipTap, Quill, Lexical, Slate) with the Bold / Italic / Underline / Link / Image buttons maps directly to this payload as HTML like:

```html
<p>This is <strong>bold</strong>, <em>italic</em>, <u>underlined</u>, and a <a href="https://example.com">link</a>.</p>
<p><img src="https://res.cloudinary.com/.../inline.png" alt="diagram" /></p>
```

### Image insertion flow (already documented above)

1. User clicks the image button in the editor.
2. Frontend opens a file picker.
3. POST to `/api/v1/uploads/image?purpose=blog`.
4. Insert `<img src="<returned secure_url>" />` at the cursor.

### Sanitisation — IMPORTANT, please confirm before merging

**The backend does NOT sanitise HTML before storing or before returning it.** It accepts whatever the editor produces and returns the same string to readers.

This matches how every other resource in this backend works (programs, universities, colleges, counselling inquiries all accept HTML/rich text and return it without sanitisation).

Because `body_html` is untrusted content (any logged-in admin can author it, and stored content can be injected with malicious HTML by a compromised admin account), **the public site MUST sanitise the string before injecting it into the DOM.** The recommended approach is [DOMPurify](https://github.com/cure53/DOMPurify), called on whichever side last touches the HTML before it reaches the browser.

#### Example (browser-side, before injecting into the DOM)

```js
import DOMPurify from 'dompurify';

// Sanitise once at the data boundary
const clean = DOMPurify.sanitize(blog.body_html, {
  ALLOWED_TAGS: ['p','strong','em','u','a','img','br','h1','h2','h3','ul','ol','li','blockquote','figure','figcaption'],
  ALLOWED_ATTR: ['href','src','alt','title','target','rel'],
});

// Force external links to open safely
const hooked = clean.replace(/<a /g, '<a target="_blank" rel="noopener noreferrer" ');

// Then inject `hooked` via your framework's HTML-binding API.
```

#### Example (server-side rendering)

For SSR/Next.js use `isomorphic-dompurify` so the same sanitisation runs on the server before the HTML hits the wire.

#### Editor-side hardening

Most modern editors (TipTap, Quill, Lexical) strip dangerous constructs by default — `<script>`, `on*` event handlers, `javascript:` URLs. Confirm your editor's default allow-list matches what's described above; if not, configure it explicitly so admins can't even inject tags that would be stripped later.

**Action required from frontend team:** confirm which rendering approach the public site uses (framework + HTML-binding API) and add DOMPurify — server-side or browser-side, whichever runs last. Reach out before merging if there's any doubt. This is the only security-relevant decision in this integration.

---

## Curl Examples (End-to-End)

Replace `/tmp/c.txt` with a cookie file captured from a real login.

```bash
# Login as admin
curl -s -i -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" -c /tmp/c.txt \
  -d '{"email":"admin@example.com","password":"..."}'
# (saves cookie to /tmp/c.txt)

# Create a draft (with author + tags)
curl -s -X POST http://localhost:8080/api/v1/blogs \
  -H "Content-Type: application/json" -b /tmp/c.txt \
  -d '{
    "title":             "Hello World",
    "slug":              "hello-world",
    "meta_description":  "A first post",
    "body_html":         "<p>Hi there!</p>",
    "cover_image":       "https://res.cloudinary.com/.../cover.png",
    "author_name":       "Jane Doe",
    "author_title":      "Editor",
    "author_description":"Writes about onboarding tips.",
    "status":            "draft"
  }'

# Upload an inline image (from the editor)
curl -s -X POST "http://localhost:8080/api/v1/uploads/image?purpose=blog" \
  -b /tmp/c.txt -F "file=@./image.png"
# → { "data": { "secure_url": "...", ... } }

# Update a draft (content + author + tags; status untouched)
curl -s -X PUT http://localhost:8080/api/v1/blogs/<id> \
  -H "Content-Type: application/json" -b /tmp/c.txt \
  -d '{
    "title": "Hello World v2",
    "slug":  "hello-world",
    "body_html": "<p>Edited</p>",
    "meta_description": "updated",
    "author_name": "Jane Doe",
    "tags": ["announcements", "editors-picks"]
  }'

# Publish
curl -s -X POST http://localhost:8080/api/v1/blogs/<id>/publish \
  -H "Content-Type: application/json" -b /tmp/c.txt \
  -d '{"publish": true}'

# Admin: list drafts (with optional tag filter)
curl -s -b /tmp/c.txt "http://localhost:8080/api/v1/blogs/all?status=draft"
curl -s -b /tmp/c.txt "http://localhost:8080/api/v1/blogs/all?status=draft&tags=admissions"

# Public: list published (with optional tag filter)
curl -s http://localhost:8080/api/v1/blogs
curl -s "http://localhost:8080/api/v1/blogs?tags=admissions,campus%20life"
curl -s "http://localhost:8080/api/v1/blogs?tags=admissions&tags=campus%20life"

# Public: get one post (no cookie)
curl -s http://localhost:8080/api/v1/blogs/hello-world

# Delete
curl -s -X DELETE http://localhost:8080/api/v1/blogs/<id> -b /tmp/c.txt
```

---

## Local Development Notes

- Backend dev server runs on `http://localhost:8080` (or whatever `PORT` is set to).
- The `blogs` table is auto-created by the migration runner on server startup — no manual setup. Author + tags columns were added in migration `20260831000001_add_author_and_tags_to_blogs.up.sql`.
- Image uploads land in Cloudinary under the folder `fmu/<APP_ENV>/blog/`. `APP_ENV=development` is the default in `.env`.
- For local testing without real Cloudinary credentials, swap `CLOUDINARY_*` env vars with sandbox credentials (any Cloudinary account works).