# API Reference

Backend API for the FMU (Find My University) application. This document covers every endpoint the frontend needs to integrate with.

- **Base URL (dev):** `http://localhost:3000`
- **All endpoints are prefixed with:** `/api/v1`
- **Content type:** `application/json` for all request and response bodies
- **Auth:** JWT bearer tokens (except where marked public)

---

## Table of contents

1. [Quick start](#quick-start)
2. [Response envelope](#response-envelope)
3. [Authentication](#authentication)
4. [Roles & permissions](#roles--permissions)
5. [Pagination](#pagination)
6. [Validation errors](#validation-errors)
7. [HTTP status codes](#http-status-codes)
8. [Auth endpoints](#auth-endpoints)
9. [University endpoints](#university-endpoints)
10. [Lookup reference data](#lookup-reference-data)

---

## Quick start

```bash
# 1. Register a user
curl -X POST http://localhost:3000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"full_name":"Ada Lovelace","email":"ada@example.com","password":"correct-horse-battery-staple"}'

# 2. Log in to get tokens
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com","password":"correct-horse-battery-staple"}'

# 3. Use the access token on protected endpoints
curl http://localhost:3000/api/v1/universities \
  -H "Authorization: Bearer <access_token>"
```

---

## Response envelope

**Every** response (success or failure) uses this shape:

```json
{
  "success": true,
  "data": { ... }
}
```

| Field      | Type            | When present                                   |
|------------|-----------------|------------------------------------------------|
| `success`  | `boolean`       | Always                                         |
| `data`     | `any`           | On success — payload depends on the endpoint   |
| `error`    | `string`        | On failure — short human-readable message       |
| `errors`   | `ErrorDetail[]` | On validation failure — per-field problem list |

**Success example:**
```json
{
  "success": true,
  "data": { "id": "...", "name": "MIT" }
}
```

**Generic error example:**
```json
{
  "success": false,
  "error": "university not found"
}
```

**Validation error example:**
```json
{
  "success": false,
  "errors": [
    { "field": "Name", "message": "Name is required" },
    { "field": "Website", "message": "Website must be a valid URL" }
  ]
}
```

`ErrorDetail` shape:
```json
{ "field": "Email", "message": "Email must be a valid email address" }
```

---

## Authentication

The API uses two tokens issued at login:

| Token          | Format                                | TTL  | Purpose                                       |
|----------------|---------------------------------------|------|-----------------------------------------------|
| `access_token` | JWT (HS256)                           | 60m  | Sent in `Authorization: Bearer <token>` header|
| `refresh_token`| Opaque random string (stored hashed)  | 168h | Used to mint a new access token               |

**Sending an authenticated request:**
```http
GET /api/v1/universities HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

The access token contains these claims (decodable for UI display, but never trust client-side for auth decisions):
```json
{
  "user_id": "d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b",
  "email": "ada@example.com",
  "role": "admin",
  "exp": 1719612345,
  "iat": 1719608745,
  "nbf": 1719608745
}
```

**401 cases (token rejected):**
- Missing `Authorization` header
- Header doesn't start with `Bearer `
- Token signature invalid / expired
- (User may still exist — token might just be stale)

When you get a 401, call `/api/v1/auth/refresh` with the refresh token to get a new access token. If that also fails, send the user back to the login screen.

---

## Roles & permissions

There are two roles:

| Role     | Default? | Can read universities | Can create universities |
|----------|----------|-----------------------|-------------------------|
| `student`| Yes (assigned at registration) | Yes | No |
| `admin`  | No (must be granted manually) | Yes | Yes |

**Promoting a user to admin** (currently done via SQL — no public endpoint):

```sql
UPDATE users SET role = 'admin' WHERE email = 'you@example.com';
```

The change takes effect on the user's next login (existing access tokens still say `student`).

**Authorization errors:**

| Status | Meaning                                                                |
|--------|------------------------------------------------------------------------|
| `401`  | No token / invalid token — user must log in                            |
| `403`  | Authenticated but role not allowed for this action                     |

---

## Pagination

List endpoints accept two query parameters and return paginated results.

**Query params:**

| Param       | Type | Default | Min | Max | Notes                                          |
|-------------|------|---------|-----|-----|------------------------------------------------|
| `page`      | int  | `1`     | 1   | —   | 1-indexed                                       |
| `page_size` | int  | `20`    | 1   | 100 | Values above 100 are silently capped to 100     |

Invalid or missing values fall back to defaults silently.

**Response shape:**
```json
{
  "success": true,
  "data": {
    "items": [ { ... }, { ... } ],
    "meta": {
      "page": 1,
      "page_size": 20,
      "total": 247,
      "total_pages": 13
    }
  }
}
```

- `items` — array of records for this page (may be empty)
- `meta.total` — total count across all pages (always the full count, not just this page)
- `meta.total_pages` — ceil(total / page_size)
- To get the next page, increment `page` by 1; if `page > total_pages` you've gone past the end

---

## Validation errors

When request body validation fails, you'll get HTTP 400 with:

```json
{
  "success": false,
  "errors": [
    { "field": "Name", "message": "Name is required" },
    { "field": "TuitionMin", "message": "TuitionMin must be at least 0" },
    { "field": "GalleryImages[2]", "message": "GalleryImages[2] must be a valid URL" }
  ]
}
```

Array fields report the index in the field name, e.g. `GalleryImages[2]` for the third image.

**Common validation rules:**

| Tag        | Meaning                                                  |
|------------|----------------------------------------------------------|
| `required` | Field must be present and non-empty                      |
| `omitempty`| Skip validation if the field is empty / zero             |
| `email`    | Must be a valid email format                             |
| `url`      | Must be a valid URL (`http://` or `https://`)            |
| `uuid`     | Must be a valid UUID                                     |
| `min`, `max` | For strings: length bounds. For numbers: value bounds. For arrays: length bounds. |
| `gte`, `lte` | Greater/less than or equal (numeric)                  |
| `dive`     | Apply the next tag to every element of a slice           |

---

## HTTP status codes

| Status | Meaning                                                                          |
|--------|----------------------------------------------------------------------------------|
| `200`  | Success                                                                          |
| `201`  | Resource created (used for `POST /api/v1/universities`)                          |
| `400`  | Bad request — malformed body or validation failure                               |
| `401`  | Unauthenticated — missing or invalid token                                       |
| `403`  | Authenticated but not allowed (e.g. student trying to create a university)       |
| `404`  | Resource not found                                                               |
| `409`  | Conflict (e.g. duplicate slug, email already registered)                         |
| `500`  | Server error — try again, contact backend if persistent                          |

---

## Auth endpoints

### POST `/api/v1/auth/register`

Create a new account. New users are always assigned the `student` role.

**Auth:** public

**Request body:**
```json
{
  "full_name": "Ada Lovelace",
  "email": "ada@example.com",
  "password": "correct-horse-battery-staple"
}
```

| Field       | Type   | Rules                       |
|-------------|--------|-----------------------------|
| `full_name` | string | required, 2–100 chars       |
| `email`     | string | required, valid email       |
| `password`  | string | required, min 8 chars       |

**Response:** `201 Created`
```json
{
  "success": true,
  "data": {
    "user_id": "d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b",
    "full_name": "Ada Lovelace",
    "email": "ada@example.com",
    "role": "student",
    "created_at": "2026-06-28T10:55:59Z"
  }
}
```

> ⚠️ This response does **not** include tokens. The user must then call `/api/v1/auth/login` to receive them.

**Errors:**
- `400` — invalid body or validation failure
- `409` — email already registered

---

### POST `/api/v1/auth/login`

Exchange email + password for an access token and a refresh token.

**Auth:** public

**Request body:**
```json
{
  "email": "ada@example.com",
  "password": "correct-horse-battery-staple"
}
```

| Field      | Type   | Rules                  |
|------------|--------|------------------------|
| `email`    | string | required, valid email  |
| `password` | string | required, min 6 chars  |

**Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "d4e5f6a7b8c9...",
    "user_id": "d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b",
    "full_name": "Ada Lovelace",
    "email": "ada@example.com",
    "avatar": "https://cdn.example.com/avatars/ada.png"
  }
}
```

**Storage recommendation (frontend):**
- `access_token` — keep in memory (or sessionStorage). Send as `Authorization: Bearer ...` on every request.
- `refresh_token` — keep in localStorage. Use only when access token expires (401 response).
- `user_id`, `full_name`, `email`, `avatar` — show in UI as needed.

**Errors:**
- `400` — invalid body or validation failure
- `401` — invalid credentials

---

### POST `/api/v1/auth/refresh`

Get a new access token using a refresh token. The old refresh token is invalidated and a new one is returned (rotation).

**Auth:** public

**Request body:**
```json
{
  "refresh_token": "d4e5f6a7b8c9..."
}
```

| Field           | Type   | Rules     |
|-----------------|--------|-----------|
| `refresh_token` | string | required  |

**Response:** `200 OK` — same shape as login.

**Errors:**
- `400` — invalid body
- `401` — refresh token is invalid, expired, or revoked

**Recommended flow on 401:**
1. Catch the 401 from any protected endpoint
2. Call `/auth/refresh` with the stored refresh token
3. If success → retry the original request with the new access token
4. If 401 again → user must log in from scratch

---

### GET `/api/v1/auth/google`

Start the Google OAuth login flow. Redirects the user to Google's consent screen.

**Auth:** public

**Response:** `302 Found` redirect to `https://accounts.google.com/...`. The browser follows this automatically.

After consent, Google redirects back to `/api/v1/auth/google/callback?code=...` which mints tokens.

---

### GET `/api/v1/auth/google/callback`

OAuth callback — completes the Google login. The frontend should not call this directly; it's hit via the redirect from Google.

**Auth:** public

**Query params:** `code` (required, supplied by Google)

**Response:** `200 OK` — same shape as login (`{access_token, refresh_token, user_id, ...}`)

**Errors:**
- `400` — missing `code` param
- `409` — email already registered with password login (must log in with password instead)

---

### DELETE `/api/v1/auth/logout`

Invalidate a refresh token (logs the user out of the current session).

**Auth:** public

**Request body:**
```json
{
  "refresh_token": "d4e5f6a7b8c9..."
}
```

**Response:** `200 OK`
```json
{ "success": true, "data": null }
```

After logout, the access token still works until it expires (max 60m), but the refresh token is dead and can't mint new ones. For full immediate invalidation, also drop the access token from memory on the client side.

---

## University endpoints

### GET `/api/v1/universities`

List universities with pagination and filtering. Returns a slim payload — for full details use the detail endpoint.

**Auth:** public

#### Pagination

| Param       | Type | Default | Notes                              |
|-------------|------|---------|------------------------------------|
| `page`      | int  | `1`     | 1-indexed                          |
| `page_size` | int  | `20`    | Max 100 (silently capped)          |

#### Filters

All filter params are optional and combine with AND across categories. Inside a multi-value param (`degree_levels=foo,bar`), values combine with OR ("school has at least one of these"). Unknown values are silently dropped rather than rejected.

**Scalar & range (academics)**
| Param | Type | Behavior |
|-------|------|----------|
| `degree_levels` | csv slug | Slug → DB name. ANY-match. Slugs: `certificate`, `associate`, `bachelors`, `masters`, `doctorate` |
| `majors` | csv slug | Slug → DB name. ANY-match. Slugs: `computer-science`, `business`, `engineering`, `medicine`, `biology`, `psychology`, `economics`, `art-design`, `law`, `nursing` |
| `study_formats` | csv slug | Slug → DB name. ANY-match. Slugs: `in-person`, `online`, `hybrid` |

**Tuition & financial aid**
| Param | Type | Behavior |
|-------|------|----------|
| `tuitionMin` | int | `tuition_min >= tuitionMin` (school's cheapest option is at least $X) |
| `tuitionMax` | int | `tuition_max <= tuitionMax` (school's most expensive option is at most $Y) |
| `offers_need_based_aid` | bool | True=filter to `need_based_aid = true` |
| `offers_merit_scholarships` | bool | True=filter to `merit_scholarships = true` |
| `no_application_fee` | bool | True=filter to `no_application_fee = true` |

> Combine both bounds to express "schools whose published range fits within $X–$Y". Universities with `NULL` on the relevant column are excluded.

**Location**
| Param | Type | Behavior |
|-------|------|----------|
| `country` | string | Exact match (case-sensitive) |
| `state_province` | string | Exact match on `state` (case-sensitive) |
| `city` | string | Exact match on `city` (case-sensitive) |

**Campus & setting**
| Param | Type | Behavior |
|-------|------|----------|
| `institution_type` | slug | Slug → DB name. Single-value. Slugs: `public`, `private-nonprofit`, `private-for-profit`, `2-year`, `4-year` |
| `campus_setting` | csv slug | Slug → DB name. ANY-match. Slugs: `urban`, `suburban`, `rural` |
| `on_campus_housing` | bool | True=filter to `on_campus_housing = true` |

**Admissions**
| Param | Type | Behavior |
|-------|------|----------|
| `acceptanceMin` | float | `acceptance_rate >= acceptanceMin` (percentage, 0–100) |
| `acceptanceMax` | float | `acceptance_rate <= acceptanceMax` |
| `testing_policy` | slug | Slug → DB name. Single-value. Slugs: `test-optional`, `test-blind`, `test-required` |

**Student life**
| Param | Type | Behavior |
|-------|------|----------|
| `special_affiliations` | csv slug | Slug → DB name. ANY-match. Slugs: `hbcu`, `hsi`, `womens-college`, `mens-college` |
| `athletics` | csv slug | Slug → DB name. ANY-match. Slugs: `ncaa-d1`, `ncaa-d2`, `ncaa-d3`, `naia`, `intramural` |
| `has_greek_life` | bool | True=school has "Greek Life" support service attached |
| `has_rotc` | bool | True=school has "ROTC" support service attached |
| `has_veteran` | bool | True=school has "Veteran Services" attached |
| `has_disability` | bool | True=school has "Disability Services" attached |
| `has_lgbtq` | bool | True=school has "LGBTQ+ Support" attached |
| `has_intl` | bool | True=school has "International Student Center" attached |

> Multiple `has_*` filters AND together — the school must have every selected support service.

**Currently not supported (frontend may send these; backend ignores them):**

- `pace_options` — no `pace` field on universities yet.
- `size` — DB stores free-text `campus_size` like "168 acres", not bucketed small/medium/large.

#### Example

```bash
# Privates in urban California that are popular, with ROTC and Computer Science, tuition ≥ $50k
curl 'http://localhost:3000/api/v1/universities?institution_type=private-nonprofit&campus_setting=urban&state_province=California&is_popular=true&has_rotc=true&majors=computer-science&tuitionMin=50000'
```

**Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "447a6419-b384-4433-b468-4692d08da4f2",
        "name": "Stanford University",
        "slug": "stanford",
        "country": "United States",
        "state": "California",
        "city": "Stanford",
        "logo": "https://cdn.example.com/stanford-logo.png",
        "cover_image": "https://cdn.example.com/stanford-cover.jpg",
        "institution_type": "Private",
        "campus_setting": "Suburban",
        "tuition_min": 56169,
        "tuition_max": 56169,
        "acceptance_rate": 3.9,
        "is_popular": true,
        "is_featured": true
      }
    ],
    "meta": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_pages": 1
    }
  }
}
```

> `acceptance_rate` is expressed as a percentage (0–100), not a fraction.
> Filter values are translated server-side; the API response is unchanged regardless of which filter slugs you send.

---

### GET `/api/v1/universities/search`

Typo-tolerant search across `name`, `city`, `state`, `country`, and `full_location`. Backed by Postgres `pg_trgm` similarity + GIN trigram indexes; results are ranked by similarity score and capped at 50.

**Auth:** public

**Query params:**

| Param | Type   | Required | Notes                                                                                |
|-------|--------|----------|--------------------------------------------------------------------------------------|
| `q`   | string  | yes      | Free-text search term. Min length 1 after trim. Max 200 chars. Typo-friendly.        |

**Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "6ead1892-d71b-4966-9ce0-d2419db9cca6",
        "name": "Massachusetts Institute of Technology",
        "slug": "mit",
        "country": "US",
        "state": "MA",
        "city": "Cambridge",
        "full_location": "Cambridge, MA, US",
        "logo": "https://cdn.example.com/mit-logo.png"
      }
    ]
  }
}
```

**Errors:**

| Status | Cause                                    | Body                                          |
|--------|------------------------------------------|-----------------------------------------------|
| `400`  | `q` missing or empty                     | `{"success": false, "error": "query parameter 'q' is required"}` |
| `400`  | `q` longer than 200 chars                | `{"success": false, "error": "query too long"}`                   |

**Examples**
```
GET /api/v1/universities/search?q=cambrige
  → returns "Cambridge University" and other near-spelling matches
GET /api/v1/universities/search?q=mit
  → returns MIT, Midwestern Institute of Technology, universities with "MA" in location
```

> The same substring hits the GIN trigram index — no full table scans. Threshold is `similarity(...) > 0.2`, tunable in `internal/db/queries/universities.sql`.

### GET `/api/v1/universities/{id}`

Get one university's full details, including all lookup-table references (majors, degree levels, study formats, etc.).

**Auth:** public

**Path params:**

| Param | Type | Notes       |
|-------|------|-------------|
| `id`  | UUID | University's primary key |

**Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "id": "6ead1892-d71b-4966-9ce0-d2419db9cca6",
    "name": "Massachusetts Institute of Technology",
    "slug": "mit",
    "overview": "MIT is a private research university in Cambridge, Massachusetts, founded in 1861.",
    "excerpt": "World-class research university.",
    "country": "US",
    "state": "MA",
    "city": "Cambridge",
    "full_location": "Cambridge, MA, US",
    "zipcode": "02139",
    "cover_image": "https://cdn.example.com/mit-cover.jpg",
    "logo": "https://cdn.example.com/mit-logo.png",
    "institution_type": "Private",
    "campus_setting": "Urban",
    "in_state_tuition": 57590,
    "out_of_state_tuition": 57590,
    "international_tuition": 57590,
    "tuition_min": 57590,
    "tuition_max": 57590,
    "need_based_aid": true,
    "merit_scholarships": true,
    "work_study": true,
    "no_application_fee": false,
    "acceptance_rate": 4.3,
    "testing_policy": "Optional",
    "sat_range": "1500-1570",
    "act_range": "34-36",
    "on_campus_housing": true,
    "freshmen_required_on_campus": false,
    "contact_email": "admissions@mit.edu",
    "contact_phone": "+1-617-253-1000",
    "website": "https://www.mit.edu",
    "avg_high_school_gpa": 4.0,
    "founded_year": 1861,
    "campus_size": "168 acres",
    "gallery_images": [
      "https://cdn.example.com/mit-1.jpg",
      "https://cdn.example.com/mit-2.jpg"
    ],
    "is_popular": true,
    "is_featured": true,
    "created_at": "2026-06-30T13:12:24.915082+05:45",
    "updated_at": "2026-06-30T13:12:24.915082+05:45",

    "degree_levels": [
      { "id": "43e1eef0-286b-4f6f-aeea-4edb72479e61", "name": "Bachelor's" },
      { "id": "0ca83a88-ad90-428a-8508-9bb30f910731", "name": "Master's" }
    ],
    "majors": [
      { "id": "7154ecda-3efe-4f2b-ae56-0b34dba16b93", "name": "Computer Science" },
      { "id": "2ee62613-5f63-469d-b5d9-bb59f27a6c50", "name": "Engineering" }
    ],
    "study_formats": [
      { "id": "a0336dd8-1e29-4564-b675-474dac1f6517", "name": "Hybrid / Blended" }
    ],
    "special_affiliations": [],
    "athletics": [
      { "id": "fa19a9f6-d650-4d85-a873-514f197b07b5", "name": "NCAA Division I" }
    ],
    "support_services": [
      { "id": "90f07d1d-2026-44a4-b47b-6b83ca15b282", "name": "International Student Center" }
    ]
  }
}
```

**Field reference:**

| Field                     | Type             | Notes                                              |
|---------------------------|------------------|----------------------------------------------------|
| `id`                      | UUID             | Primary key                                        |
| `name`, `slug`            | string           | Always present                                     |
| `overview`, `excerpt`     | string           | Empty string if not set                            |
| `country`, `state`, `city`, `full_location`, `zipcode` | string | Empty string if not set                |
| `cover_image`, `logo`     | string (URL)     | Empty string if not set                            |
| `institution_type`        | string           | e.g. "Private", "Public"                           |
| `campus_setting`          | string           | e.g. "Urban", "Suburban", "Rural"                  |
| `in_state_tuition`, `out_of_state_tuition`, `international_tuition` | number (USD/year) | `0` if not set                  |
| `tuition_min`, `tuition_max` | number         | Overall range, `0` if not set                      |
| `need_based_aid`, `merit_scholarships`, `work_study`, `no_application_fee` | bool | —                            |
| `acceptance_rate`         | number           | Percentage (0–100), not fraction                   |
| `testing_policy`          | string           | e.g. "Optional", "Required"                        |
| `sat_range`, `act_range`  | string           | e.g. "1500-1570", "34-36"                          |
| `on_campus_housing`, `freshmen_required_on_campus` | bool | —                                       |
| `contact_email`, `contact_phone`, `website` | string | Empty string if not set                    |
| `avg_high_school_gpa`     | number           | `0` if not set                                     |
| `founded_year`            | number           | `0` if not set                                     |
| `campus_size`             | string           | Empty string if not set                            |
| `gallery_images`          | string[] (URLs)  | Empty array if not set                             |
| `is_popular`, `is_featured` | bool           | Always present, default false                      |
| `created_at`, `updated_at` | string (RFC3339) | Always present                                   |
| `degree_levels`, `majors`, `study_formats`, `special_affiliations`, `athletics`, `support_services` | `[{id, name}]` | Empty array if none |

**Errors:**
- `404` — university with that ID does not exist
  ```json
  { "success": false, "error": "university not found" }
  ```

---

### POST `/api/v1/universities`

Create a new university.

**Auth:** admin only (must send `Authorization: Bearer <admin_jwt>`)

**Request body:**
```json
{
  "name": "Massachusetts Institute of Technology",
  "slug": "mit",
  "overview": "MIT is a private research university...",
  "country": "US",
  "city": "Cambridge",
  "institution_type": "Private",
  "campus_setting": "Urban",
  "contact_email": "admissions@mit.edu",
  "website": "https://www.mit.edu",

  "excerpt": "World-class research university.",
  "state": "MA",
  "full_location": "Cambridge, MA, US",
  "zipcode": "02139",
  "contact_phone": "+1-617-253-1000",
  "cover_image": "https://cdn.example.com/mit-cover.jpg",
  "logo": "https://cdn.example.com/mit-logo.png",
  "in_state_tuition": 57590,
  "out_of_state_tuition": 57590,
  "international_tuition": 57590,
  "tuition_min": 57590,
  "tuition_max": 57590,
  "need_based_aid": true,
  "merit_scholarships": true,
  "work_study": true,
  "no_application_fee": false,
  "acceptance_rate": 4.3,
  "testing_policy": "Optional",
  "sat_range": "1500-1570",
  "act_range": "34-36",
  "on_campus_housing": true,
  "freshmen_required_on_campus": false,
  "avg_high_school_gpa": 4.0,
  "founded_year": 1861,
  "campus_size": "168 acres",
  "gallery_images": [
    "https://cdn.example.com/mit-1.jpg",
    "https://cdn.example.com/mit-2.jpg"
  ],
  "is_popular": true,
  "is_featured": true,

  "degree_level_ids": ["43e1eef0-286b-4f6f-aeea-4edb72479e61"],
  "major_ids": ["7154ecda-3efe-4f2b-ae56-0b34dba16b93"],
  "study_format_ids": ["a0336dd8-1e29-4564-b675-474dac1f6517"],
  "special_affiliation_ids": [],
  "athletic_ids": ["fa19a9f6-d650-4d85-a873-514f197b07b5"],
  "support_service_ids": ["90f07d1d-2026-44a4-b47b-6b83ca15b282"]
}
```

**Field validation rules:**

**Required** (returns 400 if missing):
- `name` (string)
- `slug` (string, must be unique)
- `overview` (string)
- `country` (string)
- `city` (string)
- `institution_type` (string)
- `campus_setting` (string)
- `contact_email` (valid email)
- `website` (valid URL)
- `degree_level_ids` (array, min 1, each item must be a valid UUID)
- `major_ids` (array, min 1, each item must be a valid UUID)

**Optional, validated if present:**
- `cover_image`, `logo`, `gallery_images[]` — must be valid URLs
- `in_state_tuition`, `out_of_state_tuition`, `international_tuition`, `tuition_min`, `tuition_max` — must be ≥ 0
- `acceptance_rate` — between 0 and 100 (percentage)
- `avg_high_school_gpa` — between 0 and 5
- `founded_year` — between 1000 and 2100
- `excerpt` — max 500 chars
- `study_format_ids`, `special_affiliation_ids`, `athletic_ids`, `support_service_ids` — each item must be UUID (arrays can be empty or omitted)

**Optional, no extra validation:**
- `state`, `full_location`, `zipcode`, `contact_phone`, `campus_size`, `testing_policy`, `sat_range`, `act_range`

**Booleans (always present, default false):**
- `need_based_aid`, `merit_scholarships`, `work_study`, `no_application_fee`, `on_campus_housing`, `freshmen_required_on_campus`, `is_popular`, `is_featured`

**Response:** `201 Created` — full university record (same shape as the detail endpoint).

**Errors:**
- `400` — validation failure (per-field list in `errors`)
- `400` — one or more lookup IDs don't exist:
  ```json
  {
    "success": false,
    "errors": [
      { "field": "major_ids", "message": "the following majors do not exist: [abc-123, def-456]" }
    ]
  }
  ```
- `401` — missing or invalid token
- `403` — authenticated but role is not `admin`
- `409` — slug already taken:
  ```json
  { "success": false, "error": "university with this slug already exists (slug=mit)" }
  ```

---

## Lookup reference data

These endpoints return reference data the frontend can cache locally (it rarely changes). Each single-list endpoint returns `{ items: [{ id, name }] }`. The bundled `/lookups` endpoint returns all six lists in one object.

| Endpoint                                       | Returns               |
|------------------------------------------------|-----------------------|
| `GET /api/v1/universities/majors`              | All majors            |
| `GET /api/v1/universities/degree-levels`       | All degree levels     |
| `GET /api/v1/universities/study-formats`       | All study formats     |
| `GET /api/v1/universities/special-affiliations`| All affiliations      |
| `GET /api/v1/universities/athletics`           | All athletic divisions|
| `GET /api/v1/universities/support-services`    | All support services  |
| `GET /api/v1/universities/lookups`             | All six lists, bundled |

**All auth:** public

**Single-list example:**
```bash
curl http://localhost:3000/api/v1/universities/majors
```
```json
{
  "success": true,
  "data": {
    "items": [
      { "id": "125479fb-fccb-43cc-980a-84e1d73117b3", "name": "Art & Design" },
      { "id": "e6554eb7-068d-42cd-9255-ab810de531a9", "name": "Biology" },
      { "id": "7154ecda-3efe-4f2b-ae56-0b34dba16b93", "name": "Computer Science" }
    ]
  }
}
```

**Bundled example:**
```bash
curl http://localhost:3000/api/v1/universities/lookups
```
```json
{
  "success": true,
  "data": {
    "majors": [ { "id": "...", "name": "Computer Science" } ],
    "degree_levels": [ { "id": "...", "name": "Bachelor's" } ],
    "study_formats": [ { "id": "...", "name": "Hybrid / Blended" } ],
    "special_affiliations": [ { "id": "...", "name": "HBCU" } ],
    "athletics": [ { "id": "...", "name": "NCAA Division I" } ],
    "support_services": [ { "id": "...", "name": "International Student Center" } ]
  }
}
```

**Recommended frontend caching strategy:** fetch `/lookups` once on app load, store in a JS map keyed by ID. The detail endpoint already includes the resolved `name` for every lookup, so the cache is mainly useful when building forms (create university admin UI).

---

## Putting it all together — frontend integration checklist

- [ ] On app load: fetch `/api/v1/universities/lookups` and cache
- [ ] On login/register: store `access_token` in memory + `refresh_token` in localStorage
- [ ] Wrap `fetch`/`axios` to auto-attach `Authorization: Bearer <token>`
- [ ] Wrap `fetch`/`axios` to retry once on 401 using `/auth/refresh`; on second 401, redirect to login
- [ ] University list page: `GET /universities?page=N&page_size=20&<filters>`, render items + meta. See "Filters" in the List endpoint above — filter param names match the FilterSidebar's `searchParams` keys exactly.
- [ ] University detail page: `GET /universities/{id}`, render all fields + lookup arrays
- [ ] Admin "create university" form: 
  - Use cached lookups to populate multi-selects
  - On submit: `POST /universities` with the assembled payload
  - On 400: show the `errors[]` list inline next to each field
  - On 201: redirect to the new detail page
- [ ] Admin promotion: not a frontend concern; backend operator runs the SQL

---

## CORS

CORS is configured via the `ALLOWED_ORIGINS` env var (comma-separated list of origins, e.g. `ALLOWED_ORIGINS=http://localhost:3001`). Defaults to empty, which blocks all cross-origin requests. The middleware is mounted at the top of the chi router in `cmd/api/main.go` (`github.com/go-chi/cors`), so preflight `OPTIONS` requests are handled before any auth middleware runs. Credentials are not currently sent cross-origin; all auth endpoints return tokens in the JSON response body. If cookie-based auth is added later, set `AllowCredentials: true` in `cmd/api/main.go` and ensure the origin list does not include `*`.