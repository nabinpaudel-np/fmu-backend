# API Reference

Backend API for the FMU (Find My University) application. This document covers every endpoint the frontend needs to integrate with.

- **Base URL (dev):** `http://localhost:3000`
- **All endpoints are prefixed with:** `/api/v1`
- **Content type:** `application/json` for all request and response bodies
- **Auth:** HttpOnly session cookies (except where marked public) — see [Authentication](#authentication)

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
10. [College endpoints](#college-endpoints)
11. [Uploads endpoints](#uploads-endpoints)
12. [Favorites endpoints](#favorites-endpoints)
13. [Claim endpoints](#claim-endpoints)
14. [Lookup reference data](#lookup-reference-data)
15. [Frontend integration checklist](#putting-it-all-together--frontend-integration-checklist)
16. [CORS](#cors)

---

## Quick start

```bash
# 1. Register a user
curl -X POST http://localhost:3000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"full_name":"Ada Lovelace","email":"ada@example.com","password":"correct-horse-battery-staple"}'

# 2. Log in — response sets HttpOnly cookies
curl -c cookies.txt -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com","password":"correct-horse-battery-staple"}'

# 3. Reuse the cookie jar on subsequent requests
curl -b cookies.txt http://localhost:3000/api/v1/universities
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

The API is session-based. Logging in issues two HttpOnly cookies that the browser sends automatically on every request — JavaScript never sees the token values.

| Cookie         | Path             | TTL   | HttpOnly | SameSite | Purpose                              |
|----------------|------------------|-------|----------|----------|--------------------------------------|
| `access_token` | `/`              | 60m   | yes      | `Lax`    | JWT (HS256) carried on every request |
| `refresh_token`| `/api/v1/auth`   | 168h  | yes      | `Lax`    | Used by `/auth/refresh` to rotate    |

Both cookies are also marked `Secure` when `COOKIE_SECURE=true` (production). `HttpOnly` prevents JS access; `SameSite=Lax` blocks cross-site POSTs from sending the cookies, which is the CSRF posture (no separate CSRF token needed).

**An authenticated request — what the browser sends:**

```http
GET /api/v1/universities HTTP/1.1
Host: localhost:3000
Cookie: access_token=eyJhbGciOiJIUzI1NiIs...
```

The frontend does not read or set these cookies. The browser handles storage and attachment; the API server handles rotation and revocation. For cross-origin SPAs, the fetch/axios call must include `credentials: 'include'` (default for same-origin).

The access token is a JWT containing authorization data. A college-scoped representative token looks like:

```json
{
  "user_id": "d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b",
  "email": "ada@example.com",
  "role": "representative",
  "rep_col_id": "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10",
  "exp": 1719612345,
  "iat": 1719608745,
  "nbf": 1719608745
}
```

A university-scoped representative has `rep_uni_id` instead. Students and admins have neither field. The server uses these claims internally; because the cookie is HttpOnly, the frontend should not decode it. Call `GET /api/v1/auth/me` on app load and use the response's `representative_university_id` or `representative_college_id` field instead.

**401 cases (cookie rejected):**
- `access_token` cookie missing (user is not logged in, or cookie expired and was cleared)
- JWT signature invalid / expired

When you get a 401, call `POST /api/v1/auth/refresh` (no body — the refresh cookie is read from the request automatically). It rotates the refresh token and issues a new access token. If that also returns 401, the refresh cookie is gone/expired/revoked — send the user back to the login screen.

---

## Roles & permissions

There are three roles. A `representative` account is additionally bound to exactly one university or one college:

| Role             | How it is created | Write scope |
|------------------|-------------------|-------------|
| `student`        | Self-registration | No university or college writes |
| `representative` | Admin approves a claim | One claimed university **or** one claimed college |
| `admin`          | Granted manually | Any university or college, plus all `/api/v1/admin/*` routes |

A university-scoped representative can update their bound university and create colleges beneath it. A college-scoped representative can update only their bound college. Both can use the authenticated image upload endpoints.

**Promoting a user to admin** (currently done via SQL — no public endpoint):

```sql
UPDATE users SET role = 'admin' WHERE email = 'you@example.com';
```

The change takes effect on the user's next login (existing session cookies still carry the old `student` role until the access token expires or the user logs in again).

**Representative accounts** are not created by self-registration. An admin creates one by approving a [university or college claim](#claim-endpoints), then securely delivers the one-time plaintext password from the approval response. The login, refresh, and `/auth/me` responses identify the scope with either `representative_university_id` or `representative_college_id`. Only one representative may be bound to each target; the database enforces separate UNIQUE constraints for universities and colleges. See [Representative editing rights](#representative-editing-rights) for the complete scope rules.

**Authorization errors:**

| Status | Meaning                                                                |
|--------|------------------------------------------------------------------------|
| `401`  | No session cookie / invalid cookie — user must log in                 |
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
| `401`  | Unauthenticated — missing or invalid session cookie                             |
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

> ⚠️ This response does **not** set auth cookies. The user must then call `/api/v1/auth/login` to start a session.

**Errors:**
- `400` — invalid body or validation failure
- `409` — email already registered

---

### POST `/api/v1/auth/login`

Exchange email + password for an authenticated session. On success the response sets the `access_token` and `refresh_token` cookies and returns the user object.

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

**Response:** `200 OK` — sets cookies, returns:
```json
{
  "success": true,
  "data": {
    "user_id": "d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b",
    "full_name": "Ada Lovelace",
    "email": "ada@example.com",
    "avatar": "https://cdn.example.com/avatars/ada.png",
    "role": "representative",
    "representative_college_id": "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10"
  }
}
```

`role` is always present. Representative accounts include exactly one applicable scope field:

- University representative → `representative_university_id`
- College representative → `representative_college_id`
- Student or admin → neither field

The inapplicable field is omitted rather than returned as `null` or an empty string. Use the same rule when typing the `/auth/refresh` and `/auth/me` responses.

**Cookies set on success:**

| Cookie         | Value | Path            | Max-Age | Flags                              |
|----------------|-------|-----------------|---------|------------------------------------|
| `access_token` | JWT   | `/`             | 60m     | HttpOnly; SameSite=Lax; Secure*    |
| `refresh_token`| opaque| `/api/v1/auth`  | 168h    | HttpOnly; SameSite=Lax; Secure*    |

*`Secure` is set when `COOKIE_SECURE=true` (production). Off by default so the cookies work over `http://localhost`.

**Storage recommendation (frontend):** store no token values; the browser handles the cookies. Keep the returned user object in application state and use `role` plus the optional representative scope ID to decide which editing controls to render. Server authorization remains authoritative.

**Curl example:**
```bash
curl -c cookies.txt -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com","password":"correct-horse-battery-staple"}'
```

**Errors:**
- `400` — invalid body or validation failure
- `401` — invalid credentials

---

### POST `/api/v1/auth/refresh`

Rotate the session. Reads the `refresh_token` cookie (no request body), invalidates it server-side, and sets a fresh pair of cookies. Returns the current user object.

**Auth:** public — the `refresh_token` cookie itself is the credential.

**Request body:** none. The endpoint reads the `refresh_token` cookie from the request.

**Response:** `200 OK` — sets new cookies and returns the same user shape as login:
```json
{
  "success": true,
  "data": {
    "user_id": "d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b",
    "full_name": "Ada Lovelace",
    "email": "ada@example.com",
    "avatar": "https://cdn.example.com/avatars/ada.png",
    "role": "representative",
    "representative_college_id": "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10"
  }
}
```

The previous `refresh_token` is dead the moment the response is sent. Use the cookies from the new response on the retry.

**Errors:**
- `401` — `refresh_token` cookie missing, invalid, expired, or revoked. Both cookies are cleared on this response.

**Recommended flow on 401 from a protected endpoint:**
1. Catch the 401
2. `POST /api/v1/auth/refresh` (with `credentials: 'include'` so the cookie is sent)
3. On 200 → retry the original request (the new `access_token` cookie is now in the jar)
4. On 401 again → user must log in from scratch

---

### GET `/api/v1/auth/google`

Start the Google OAuth login flow. Redirects the user to Google's consent screen.

**Auth:** public

**Response:** `302 Found` redirect to `https://accounts.google.com/...`. The browser follows this automatically.

**Security posture:**

- The server generates a fresh random `state` value (also used as the PKCE `code_verifier`), stores it in a short-lived `oauth_state` cookie scoped to the callback URL, and adds it to the auth request along with the S256 challenge.
- `prompt=select_account` is set so users with multiple Google accounts always see the account picker.
- The `code` query parameter is **never** accepted on this endpoint. Accepting it would let an attacker trick a victim into completing a flow with the attacker's authorization code (account takeover via session fixation). To re-run the flow, send the user here with no query string.

After consent, Google redirects back to `/api/v1/auth/google/callback?code=...&state=...`, which validates the state, exchanges the code, sets the auth cookies, and 302s the user to the frontend.

---

### GET `/api/v1/auth/google/callback`

OAuth callback — completes the Google login. The frontend should not call this directly; it's hit via the redirect from Google.

**Auth:** public

**Query params (supplied by Google):**

| Param        | Required | Notes                                              |
|--------------|----------|----------------------------------------------------|
| `code`       | yes      | Authorization code (unless `error` is set)         |
| `state`      | yes      | Must match the value stored in the `oauth_state` cookie at flow initiation. CSRF check. |
| `error`      | no       | If present, the flow failed at the provider (e.g. `access_denied`, `invalid_scope`) |
| `error_description` | no | Human-readable error detail from the provider      |

**Response on success:** `302 Found` redirect to `FRONTEND_URL` (e.g. `http://localhost:3001/`). The `access_token` and `refresh_token` cookies are set on this response, and the `oauth_state` cookie is cleared. There is no JSON body and no tokens in the URL — the frontend just becomes authenticated as soon as the next API call is made.

**Response on provider error (`?error=...` from Google):** `302 Found` redirect to `FRONTEND_URL?error=<error>&error_description=<error_description>`. The `oauth_state` cookie is cleared. The frontend should display the error and prompt the user to try again.

**Response on backend validation failure:**

| Status | Cause                                                                            | Response     |
|--------|----------------------------------------------------------------------------------|--------------|
| `400`  | Missing `code` (and no `error` from provider)                                    | JSON error   |
| `401`  | `state` doesn't match the `oauth_state` cookie — CSRF check failed              | JSON error   |

**Account conflict:** if the Google account's email is already registered with password login, the response is a `302` redirect to `FRONTEND_URL?error=email_taken&error_description=this+email+is+already+registered+with+password+login`. The frontend should send the user to the password login screen.

---

### DELETE `/api/v1/auth/logout`

End the current session. Reads the `refresh_token` cookie (no request body), revokes it server-side, and clears both cookies.

**Auth:** public — the `refresh_token` cookie is the credential.

**Request body:** none.

**Response:** `200 OK`
```json
{ "success": true, "data": null }
```

On success the response also sets `access_token` and `refresh_token` cookies with `Max-Age=0` to clear them in the browser. The endpoint is idempotent — if the cookie is already missing or expired, it still returns 200 and clears whatever remains.

After logout, the server immediately rejects any further use of the old refresh token. The `access_token` cookie is also cleared, so no protected requests will succeed.

---

### GET `/api/v1/auth/me`

Return the currently-authenticated user. Use this on app load to bootstrap user state in the SPA — it's the only way to know "who am I?" from a page reload, because the auth cookies are HttpOnly and the frontend can't read the JWT directly.

**Auth:** required (must have a valid `access_token` cookie)

**Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "user_id": "d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b",
    "full_name": "Ada Lovelace",
    "email": "ada@example.com",
    "avatar": "https://cdn.example.com/avatars/ada.png",
    "role": "representative",
    "representative_college_id": "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10"
  }
}
```

A university representative receives `representative_university_id` instead. Student and admin responses omit both representative ID fields.

The data is read fresh from the database on each call (cheap — `users` PK lookup), so avatar/name/role reflect the current state, not just the JWT claims.

**Errors:**
- `401` — no `access_token` cookie, or it's invalid/expired

**Curl example:**
```bash
curl -b cookies.txt http://localhost:3000/api/v1/auth/me
```

---

## University endpoints

Reads are public. Creating a university is admin-only; updating allows an admin or the representative bound to that university.

| Endpoint                                       | Auth   |
|------------------------------------------------|--------|
| `GET /api/v1/universities`                     | public |
| `GET /api/v1/universities/search`              | public |
| `GET /api/v1/universities/stats`               | admin  |
| `GET /api/v1/universities/{id}`                | public |
| `GET /api/v1/universities/{id}/colleges`       | public |
| `GET /api/v1/universities/majors`              | public |
| `GET /api/v1/universities/degree-levels`       | public |
| `GET /api/v1/universities/study-formats`       | public |
| `GET /api/v1/universities/special-affiliations`| public |
| `GET /api/v1/universities/athletics`           | public |
| `GET /api/v1/universities/support-services`    | public |
| `GET /api/v1/universities/lookups`             | public |
| `POST /api/v1/universities`                    | admin                           |
| `PATCH /api/v1/universities/{id}`              | admin or matching university rep |

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
        "is_featured": true,
        "is_favorited": false,
        "has_representative": true
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
>
> Each item carries an `is_favorited` boolean. If the request includes a valid session cookie, items the authenticated student has favorited have `is_favorited: true`; otherwise `false`. Anonymous requests always receive `is_favorited: false`.
>
> `has_representative` is viewer-independent. It is `true` when a representative user is already bound to that university and `false` otherwise. Use it to hide the public claim banner; the same field is present on university detail and search responses.

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
        "logo": "https://cdn.example.com/mit-logo.png",
        "is_favorited": false,
        "has_representative": true
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

### GET `/api/v1/universities/stats`

Aggregate counts for the admin dashboard. One Postgres round-trip; computed via `COUNT`, `COUNT(DISTINCT country)`, and `COUNT(*) FILTER (WHERE …)` against the `universities` table.

**Auth:** admin only (requires an authenticated admin session cookie)

**Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "total_universities": 247,
    "total_countries": 12,
    "total_featured": 18,
    "total_popular": 24
  }
}
```

| Field               | Type | Notes                                            |
|---------------------|------|--------------------------------------------------|
| `total_universities`| int  | Row count of the `universities` table           |
| `total_countries`   | int  | Distinct non-null `country` values              |
| `total_featured`    | int  | Rows where `is_featured = true`                  |
| `total_popular`     | int  | Rows where `is_popular = true`                   |

**Errors:**
- `401` — missing or invalid session cookie
- `403` — authenticated but role is not `admin`

---

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
    "has_representative": true,
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
| `has_representative`       | bool           | `true` when a representative user is bound; use to hide the claim banner |
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

**Auth:** admin only (requires an authenticated admin session cookie)

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
  "cover_image": "https://res.cloudinary.com/<cloud>/image/upload/v1234567890/fmu/development/cover/abc123.jpg",
  "logo": "https://res.cloudinary.com/<cloud>/image/upload/v1234567890/fmu/development/logo/xyz789.png",
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
    "https://res.cloudinary.com/<cloud>/image/upload/v1234567890/fmu/development/gallery/img1.jpg",
    "https://res.cloudinary.com/<cloud>/image/upload/v1234567890/fmu/development/gallery/img2.jpg"
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
- `cover_image`, `logo`, `gallery_images[]` — must be valid URLs. The expected workflow is to upload first via `POST /api/v1/uploads/sign` (browser → Cloudinary direct), then pass the returned `secure_url` here. See [Uploads endpoints](#uploads-endpoints) for the two-step flow.
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
- `401` — missing or invalid session cookie
- `403` — authenticated but role is not `admin`
- `409` — slug already taken:
  ```json
  { "success": false, "error": "university with this slug already exists (slug=mit)" }
  ```

---

### PATCH `/api/v1/universities/{id}`

Partial update of an existing university. Send only the fields you want to change — anything omitted is left as-is.

**Auth:** admin, or a university-scoped representative whose `representative_university_id` equals the path `{id}`. Admins may update every accepted field. Representatives may update profile data but must not include `name` or `slug`; either key returns `403 Forbidden`.

**Path params:**

| Param | Type | Notes              |
|-------|------|--------------------|
| `id`  | UUID | University's primary key |

**Request body:** same field names and types as [POST `/api/v1/universities`](#post-apiv1universities), but every field is **optional**. Pointer types in the DTO mean:
- **Omit the field** → unchanged in the database.
- **Send `null`** → also leaves the value unchanged (JSON `null` decodes to a nil pointer in Go).
- **Send a value** → that value replaces the existing one.

To clear a string-typed column, send the empty string `""`. To clear a numeric column, omit it (there is no natural "unset" representation for numbers — sending `0` would actually update the column to `0`).

**Lookup-table arrays** (`degree_level_ids`, `major_ids`, `study_format_ids`, `special_affiliation_ids`, `athletic_ids`, `support_service_ids`) are special:
- **Omit the field** → existing associations stay.
- **Send `[]`** → all existing associations for that category are removed.
- **Send `["uuid", ...]`** → existing associations for that category are replaced with the new list.

So a `[]` payload is a *destructive* delete for that category. To "leave alone", omit the key entirely.

**Example — flip a boolean and add a major:**
```json
{
  "is_featured": true,
  "major_ids": ["7154ecda-3efe-4f2b-ae56-0b34dba16b93"]
}
```

**Example — bump tuition without touching anything else:**
```json
{
  "tuition_min": 60000,
  "tuition_max": 65000
}
```

**Example — drop all athletics affiliations:**
```json
{
  "athletic_ids": []
}
```

**Response:** `200 OK` — same shape as the create response (the slim record, no `degree_levels`, `majors`, etc. arrays). `updated_at` is set to the current time; all other untouched fields keep their existing values.
```json
{
  "success": true,
  "data": {
    "id": "6ead1892-d71b-4966-9ce0-d2419db9cca6",
    "name": "Massachusetts Institute of Technology",
    "slug": "mit",
    "overview": "MIT is a private research university...",
    "excerpt": "World-class research university.",
    "country": "US",
    "state": "MA",
    "city": "Cambridge",
    "full_location": "Cambridge, MA, US",
    "zipcode": "02139",
    "cover_image": "https://res.cloudinary.com/<cloud>/image/upload/.../cover/abc123.jpg",
    "logo": "https://res.cloudinary.com/<cloud>/image/upload/.../logo/xyz789.png",
    "institution_type": "Private",
    "campus_setting": "Urban",
    "in_state_tuition": 57590,
    "out_of_state_tuition": 57590,
    "international_tuition": 57590,
    "tuition_min": 60000,
    "tuition_max": 65000,
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
      "https://res.cloudinary.com/<cloud>/image/upload/.../gallery/img1.jpg"
    ],
    "is_popular": true,
    "is_featured": true,
    "created_at": "2026-06-30T13:12:24.915082+05:45",
    "updated_at": "2026-07-07T10:00:00+05:45"
  }
}
```

The whole operation runs inside a single transaction. If any provided lookup ID doesn't exist, the SQL `UPDATE` is rolled back and no fields change.

**Field validation rules:** same per-field validators as create (`url`, `email`, `uuid`, range bounds, etc.) — but every field is `omitempty`, so missing fields aren't checked.

**Errors:**
- `400` — invalid body or validation failure (e.g. `cover_image` isn't a URL, `acceptance_rate` is > 100)
- `400` — one or more provided lookup IDs don't exist:
  ```json
  {
    "success": false,
    "errors": [
      { "field": "major_ids", "message": "the following majors do not exist: [abc-123, def-456]" }
    ]
  }
  ```
- `401` — missing/invalid session cookie
- `403` — caller is neither an admin nor the representative bound to this university
- `403` — a bound representative supplied `name` or `slug`:
  ```json
  { "success": false, "error": "representatives cannot change name or slug" }
  ```
- `404` — university with that ID does not exist (or `id` is not a valid UUID):
  ```json
  { "success": false, "error": "university not found" }
  ```
- `409` — slug already taken by another university:
  ```json
  { "success": false, "error": "university with this slug already exists (slug=mit)" }
  ```

---

### GET `/api/v1/universities/{id}/colleges`

List colleges affiliated with a specific university. Returns a slim payload — for full details use `GET /api/v1/colleges/{id}`.

**Auth:** public

**Path params:**

| Param | Type | Notes                  |
|-------|------|------------------------|
| `id`  | UUID | Parent university's primary key |

**Query params:**

| Param       | Type | Default | Notes                              |
|-------------|------|---------|------------------------------------|
| `page`      | int  | `1`     | 1-indexed                          |
| `page_size` | int  | `20`    | Max 100 (silently capped)          |

**Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10",
        "name": "MIT College of Engineering",
        "slug": "mit-college-of-engineering",
        "university_id": "125479fb-fccb-43cc-980a-84e1d73117b3",
        "country": "US",
        "state": "MA",
        "city": "Cambridge",
        "logo": "https://res.cloudinary.com/<cloud>/image/upload/.../logo/abc.png",
        "cover_image": "https://res.cloudinary.com/<cloud>/image/upload/.../cover/abc.jpg",
        "institution_type": "Private",
        "campus_setting": "Urban",
        "is_popular": true,
        "is_featured": false,
        "is_favorited": false,
        "has_representative": true
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

Results are ordered by `name` ascending. If the parent university exists but has no affiliated colleges, `items` is `[]` and `meta.total` is `0`.

**Errors:**
- `404` — `id` is not a valid UUID, or no university with that ID exists:
  ```json
  { "success": false, "error": "university not found" }
  ```

---

## College endpoints

A college is an institution affiliated to a parent university (e.g. an engineering college under a university). The parent reference is required — every college must point at a real `universities.id`. Deleting a university while colleges still reference it is blocked by the database (`ON DELETE RESTRICT`).

Reads are public. Creating a college allows admins and university-scoped representatives; updating allows admins and the representative bound to that college.

| Endpoint                                  | Auth                              |
|-------------------------------------------|-----------------------------------|
| `GET /api/v1/colleges`                    | public                            |
| `GET /api/v1/colleges/search`             | public                            |
| `GET /api/v1/colleges/{id}`               | public                            |
| `GET /api/v1/universities/{id}/colleges`  | public                            |
| `POST /api/v1/colleges`                   | admin or matching university rep  |
| `PATCH /api/v1/colleges/{id}`             | admin or matching college rep     |

### POST `/api/v1/colleges`

Create a new college under an existing parent university.

**Auth:** admin (any university) or representative. A representative may only create colleges whose `university_id` matches their own bound university — see [Representative editing rights](#representative-editing-rights).

**Request body:**
```json
{
  "name": "MIT College of Engineering",
  "slug": "mit-college-of-engineering",
  "university_id": "125479fb-fccb-43cc-980a-84e1d73117b3",
  "overview": "The engineering undergraduate college affiliated with MIT.",
  "excerpt": "Engineering education and research in Cambridge.",
  "country": "US",
  "state": "MA",
  "city": "Cambridge",
  "full_location": "Cambridge, MA, US",
  "zipcode": "02139",
  "cover_image": "https://res.cloudinary.com/<cloud>/image/upload/v1234/fmu/development/cover/abc.jpg",
  "logo": "https://res.cloudinary.com/<cloud>/image/upload/v1234/fmu/development/logo/abc.png",
  "gallery_images": [
    "https://res.cloudinary.com/<cloud>/image/upload/v1234/fmu/development/gallery/abc-1.jpg"
  ],
  "institution_type": "Private",
  "campus_setting": "Urban",
  "contact_email": "engineering@mit.edu",
  "contact_phone": "+1-617-253-1000",
  "website": "https://engineering.mit.edu",
  "founded_year": 1932,
  "campus_size": "Cambridge campus",
  "is_popular": true,
  "is_featured": false
}
```

**Required, validated:**
- `name` — non-empty, max 255 chars
- `slug` — non-empty, 2–255 chars. Must be unique across colleges.
- `university_id` — must be a valid UUID of an existing `universities.id`
- `overview` — non-empty

**Optional, validated if present:**
- `excerpt` — max 500 chars
- `country`, `state`, `city` — max 100 chars
- `full_location` — max 255 chars; `zipcode` — max 20 chars
- `cover_image`, `logo`, and every `gallery_images[]` item — valid URL. Upload images first through `POST /api/v1/uploads/sign`, then store the returned `secure_url` values. See [Uploads endpoints](#uploads-endpoints).
- `institution_type`, `campus_setting` — max 50 chars
- `contact_email` — valid email, max 255 chars; `contact_phone` — max 50 chars
- `website` — valid URL, max 500 chars
- `founded_year` — integer from 1000 through 2100; `campus_size` — max 100 chars
- `is_popular`, `is_featured` — booleans, default `false`

**Response:** `201 Created`
```json
{
  "success": true,
  "data": {
    "id": "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10",
    "name": "MIT College of Engineering",
    "slug": "mit-college-of-engineering",
    "university_id": "125479fb-fccb-43cc-980a-84e1d73117b3",
    "overview": "The engineering undergraduate college affiliated with MIT.",
    "excerpt": "Engineering education and research in Cambridge.",
    "country": "US",
    "state": "MA",
    "city": "Cambridge",
    "full_location": "Cambridge, MA, US",
    "zipcode": "02139",
    "cover_image": "https://res.cloudinary.com/<cloud>/image/upload/v1234/fmu/development/cover/abc.jpg",
    "logo": "https://res.cloudinary.com/<cloud>/image/upload/v1234/fmu/development/logo/abc.png",
    "gallery_images": [
      "https://res.cloudinary.com/<cloud>/image/upload/v1234/fmu/development/gallery/abc-1.jpg"
    ],
    "institution_type": "Private",
    "campus_setting": "Urban",
    "contact_email": "engineering@mit.edu",
    "contact_phone": "+1-617-253-1000",
    "website": "https://engineering.mit.edu",
    "founded_year": 1932,
    "campus_size": "Cambridge campus",
    "is_popular": true,
    "is_featured": false,
    "created_at": "2026-07-06T15:00:00Z",
    "updated_at": "2026-07-06T15:00:00Z"
  }
}
```

**Errors:**
- `400` — invalid JSON, per-field validation failure, or `university_id` does not exist:
  ```json
  { "success": false, "error": "parent university does not exist (university_id=125479fb-fccb-43cc-980a-84e1d73117b3)" }
  ```
- `401` — missing/invalid session cookie
- `403` — authenticated but not permitted: role is `student`, or a `representative` whose bound university does not match the body `university_id`:
  ```json
  { "success": false, "error": "representative can only edit their own university" }
  ```
- `409` — slug already taken:
  ```json
  { "success": false, "error": "college with this slug already exists (slug=mit-college-of-engineering)" }
  ```

---

### GET `/api/v1/colleges`

List colleges with pagination and filtering. Returns a slim payload — for full details use the detail endpoint.

**Auth:** public

#### Pagination

| Param       | Type | Default | Notes                              |
|-------------|------|---------|------------------------------------|
| `page`      | int  | `1`     | 1-indexed                          |
| `page_size` | int  | `20`    | Max 100 (silently capped)          |

#### Filters

All filter params are optional and combine with AND.

| Param            | Type   | Behavior                                                            |
|------------------|--------|---------------------------------------------------------------------|
| `university_id`  | UUID   | Exact match on `university_id`. Use this to scope the list to colleges under one parent (same payload as `GET /api/v1/universities/{id}/colleges`, but available without knowing the URL up front). |
| `country`        | string | Exact match (case-sensitive)                                        |
| `state_province` | string | Exact match on `state` (case-sensitive)                             |
| `city`           | string | Exact match on `city` (case-sensitive)                              |

Unknown or empty values narrow to nothing.

**Example:**
```bash
# All colleges under MIT in Massachusetts
curl 'http://localhost:3000/api/v1/colleges?university_id=125479fb-fccb-43cc-980a-84e1d73117b3&state_province=Massachusetts'
```

**Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10",
        "name": "MIT College of Engineering",
        "slug": "mit-college-of-engineering",
        "university_id": "125479fb-fccb-43cc-980a-84e1d73117b3",
        "country": "US",
        "state": "MA",
        "city": "Cambridge",
        "logo": "https://res.cloudinary.com/<cloud>/image/upload/v1234/fmu/development/logo/abc.png",
        "cover_image": "https://res.cloudinary.com/<cloud>/image/upload/v1234/fmu/development/cover/abc.jpg",
        "institution_type": "Private",
        "campus_setting": "Urban",
        "is_popular": true,
        "is_featured": false,
        "is_favorited": false,
        "has_representative": true
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

Results are ordered by `name` ascending. College list items do **not** embed the parent university — if the UI needs the parent name in card view, fetch `GET /api/v1/universities/{university_id}` per item or use `/search` (which embeds it).

> Each item carries an `is_favorited` boolean. If the request includes a valid session cookie, items the authenticated student has favorited have `is_favorited: true`; otherwise `false`. Same rule applies to `/api/v1/colleges/search` and `/api/v1/universities/{id}/colleges`.
>
> Every college list item also includes `has_representative`. It is `true` when a representative user is bound to the college and is independent of the viewer. Hide the claim banner when this value is `true`; the same field is returned by college detail and search endpoints.

---

### GET `/api/v1/colleges/search`

Typo-tolerant search across college **and parent university** fields. Backed by Postgres `pg_trgm` similarity + a GIN trigram index on `colleges.name`; results are ranked by similarity score and capped at 50.

**Fields the query matches against** (similarity threshold `0.2` each):

| Field                          | Matched on college or parent? |
|--------------------------------|--------------------------------|
| `colleges.name`                | college                         |
| `colleges.full_location`       | college                         |
| `colleges.city`                | college                         |
| `colleges.state`               | college                         |
| `colleges.country`             | college                         |
| `universities.name`  *(parent)*| university                      |
| `universities.slug`  *(parent)*| university                      |

The endpoint joins to the parent university, so a single `q` finds both "College of Engineering at MIT" and "All colleges under Harvard". Results are ranked by `GREATEST(...)` of similarity across the fields, with `name` as the stable tiebreaker.

**Auth:** public

**Query params:**

| Param | Type   | Required | Notes                                                                                |
|-------|--------|----------|--------------------------------------------------------------------------------------|
| `q`   | string | yes      | Free-text search term. Min length 1 after trim. Max 200 chars. Typo-friendly.        |

**Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10",
        "name": "MIT College of Engineering",
        "slug": "mit-college-of-engineering",
        "university": {
          "id": "125479fb-fccb-43cc-980a-84e1d73117b3",
          "name": "Massachusetts Institute of Technology",
          "slug": "mit",
          "logo": "https://res.cloudinary.com/<cloud>/image/upload/v1234/fmu/development/universities/mit-logo.png"
        },
        "country": "US",
        "state": "MA",
        "city": "Cambridge",
        "logo": "https://res.cloudinary.com/<cloud>/image/upload/v1234/fmu/development/colleges/abc.png",
        "is_favorited": false,
        "has_representative": true
      }
    ]
  }
}
```

The `university` object is embedded so the client can render cards like "MIT College of Engineering · MIT" without a second request. Its fields are empty strings when not set.

**Errors:**

| Status | Cause                                    | Body                                          |
|--------|------------------------------------------|-----------------------------------------------|
| `400`  | `q` missing or empty                     | `{"success": false, "error": "query parameter 'q' is required"}` |
| `400`  | `q` longer than 200 chars                | `{"success": false, "error": "query too long"}`                   |

**Examples**
```
GET /api/v1/colleges/search?q=harvard
  → returns "Harvard College" and any college whose parent is "Harvard University"
  → (also matches Harvard-style typos via the trigram index)

GET /api/v1/colleges/search?q=cambrige
  → returns colleges in Cambridge and any whose parent is a Cambridge university
```

> Only `colleges.name` has a trigram index today (`idx_colleges_name_trgm`). The remaining college fields and the join to `universities` fall back to seq scans — fine while the colleges table stays small; add GIN indexes on `universities.name`/`universities.slug` if the result set grows.
>
> Each item carries an `is_favorited` boolean, populated the same way as the university list (see `GET /api/v1/universities` above).

---

---

### GET `/api/v1/colleges/{id}`

Get one college's full details.

**Auth:** public

**Path params:**

| Param | Type | Notes       |
|-------|------|-------------|
| `id`  | UUID | College's primary key |

**Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "id": "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10",
    "name": "MIT College of Engineering",
    "slug": "mit-college-of-engineering",
    "university_id": "125479fb-fccb-43cc-980a-84e1d73117b3",
    "overview": "The engineering undergraduate college affiliated with MIT.",
    "excerpt": "Engineering education and research in Cambridge.",
    "country": "US",
    "state": "MA",
    "city": "Cambridge",
    "full_location": "Cambridge, MA, US",
    "zipcode": "02139",
    "cover_image": "https://res.cloudinary.com/<cloud>/image/upload/v1234/fmu/development/cover/abc.jpg",
    "logo": "https://res.cloudinary.com/<cloud>/image/upload/v1234/fmu/development/logo/abc.png",
    "gallery_images": [
      "https://res.cloudinary.com/<cloud>/image/upload/v1234/fmu/development/gallery/abc-1.jpg"
    ],
    "institution_type": "Private",
    "campus_setting": "Urban",
    "contact_email": "engineering@mit.edu",
    "contact_phone": "+1-617-253-1000",
    "website": "https://engineering.mit.edu",
    "founded_year": 1932,
    "campus_size": "Cambridge campus",
    "is_popular": true,
    "is_featured": false,
    "has_representative": true,
    "created_at": "2026-07-06T15:00:00Z",
    "updated_at": "2026-07-06T15:00:00Z"
  }
}
```

| Field             | Type             | Notes                                       |
|-------------------|------------------|---------------------------------------------|
| `id`              | UUID             | Primary key                                 |
| `name`, `slug`    | string           | Always present                              |
| `university_id`   | UUID             | Parent university                           |
| `overview`        | string           | Always present                              |
| `excerpt`         | string           | Empty string if not set                     |
| `country`, `state`, `city`, `full_location`, `zipcode` | string | Empty string if not set |
| `cover_image`, `logo` | string (URL) | Empty string if not set                     |
| `gallery_images`  | string[] (URLs)  | College gallery images                      |
| `institution_type`, `campus_setting` | string | Empty string if not set             |
| `contact_email`, `contact_phone`, `website` | string | Empty string if not set          |
| `founded_year`    | number           | `0` if not set                              |
| `campus_size`     | string           | Empty string if not set                     |
| `is_popular`, `is_featured` | bool  | Always present, default `false`              |
| `has_representative` | bool         | `true` when a representative is bound; use to hide the claim banner |
| `created_at`, `updated_at` | string (RFC3339) | Always present                    |

**Errors:**
- `404` — college with that ID does not exist
  ```json
  { "success": false, "error": "college not found" }
  ```

---

### PATCH `/api/v1/colleges/{id}`

Partially update a college. Fetch the current record with `GET /api/v1/colleges/{id}`, send only changed fields, and replace local state with the returned record.

**Auth:** admin, or a college-scoped representative whose `representative_college_id` equals the path `{id}`. Admins may update every accepted field. Representatives may update profile data but must not include `name` or `slug`; either key returns `403 Forbidden`. A university-scoped representative cannot use this route.

**Path params:**

| Param | Type | Notes |
|-------|------|-------|
| `id`  | UUID | College's primary key |

**Request body:** every field is optional.

```json
{
  "overview": "Updated overview.",
  "city": "Cambridge",
  "contact_phone": "+1-617-253-1000",
  "website": "https://engineering.mit.edu",
  "cover_image": "https://res.cloudinary.com/<cloud>/image/upload/v1234/fmu/development/cover/new-cover.jpg"
}
```

| Field | Type | Validation |
|-------|------|------------|
| `name` | string | 1–255 chars; admin only |
| `slug` | string | 2–255 chars, unique across colleges; admin only |
| `overview` | string | min 1 char |
| `excerpt` | string | max 500 chars |
| `country`, `state`, `city` | string | max 100 chars |
| `full_location` | string | max 255 chars |
| `zipcode` | string | max 20 chars |
| `cover_image`, `logo` | string | valid URL |
| `gallery_images` | string[] | every item must be a valid URL |
| `institution_type`, `campus_setting` | string | max 50 chars |
| `contact_email` | string | valid email, max 255 chars |
| `contact_phone` | string | max 50 chars |
| `website` | string | valid URL, max 500 chars |
| `founded_year` | integer | 1000–2100 |
| `campus_size` | string | max 100 chars |
| `is_popular`, `is_featured` | boolean | no additional validation |

Partial-update rules:

- Omitted field → unchanged.
- JSON `null` → unchanged.
- Empty string for an optional location or logo field → stores an empty string, which clears it from the frontend's perspective.
- `{}` is valid and returns the current record without changing it.
- `university_id` is not accepted by this endpoint; a college cannot be moved to a different parent university.
- Admins may send `name` and `slug`. College representatives must omit both fields, even when sending the current unchanged value.

**Example:**

```bash
curl -b cookies.txt -X PATCH http://localhost:3000/api/v1/colleges/9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10 \
  -H "Content-Type: application/json" \
  -d '{"overview":"Updated overview.","contact_phone":"+1-617-253-1000","website":"https://engineering.mit.edu"}'
```

**Response:** `200 OK` — full updated college record in the same shape as the create response. Mutation responses do not include `has_representative`; preserve the value already in client state or refetch `GET /api/v1/colleges/{id}`.

```json
{
  "success": true,
  "data": {
    "id": "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10",
    "name": "MIT College of Engineering",
    "slug": "mit-college-of-engineering",
    "university_id": "125479fb-fccb-43cc-980a-84e1d73117b3",
    "overview": "Updated overview.",
    "excerpt": "Engineering education and research in Cambridge.",
    "country": "US",
    "state": "MA",
    "city": "Cambridge",
    "full_location": "Cambridge, MA, US",
    "zipcode": "02139",
    "cover_image": "https://res.cloudinary.com/<cloud>/image/upload/v1234/fmu/development/cover/abc.jpg",
    "logo": "https://res.cloudinary.com/<cloud>/image/upload/v1234/fmu/development/logo/abc.png",
    "gallery_images": [],
    "institution_type": "Private",
    "campus_setting": "Urban",
    "contact_email": "engineering@mit.edu",
    "contact_phone": "+1-617-253-1000",
    "website": "https://engineering.mit.edu",
    "founded_year": 1932,
    "campus_size": "Cambridge campus",
    "is_popular": true,
    "is_featured": false,
    "created_at": "2026-07-06T15:00:00Z",
    "updated_at": "2026-07-23T12:00:00Z"
  }
}
```

**Errors:**

- `400` — malformed JSON or field validation failure
- `401` — missing or invalid session cookie
- `403` — caller is neither an admin nor the representative bound to this college:
  ```json
  { "success": false, "error": "forbidden" }
  ```
- `403` — a bound college representative supplied `name` or `slug`:
  ```json
  { "success": false, "error": "representatives cannot change name or slug" }
  ```
- `404` — college does not exist:
  ```json
  { "success": false, "error": "college not found" }
  ```
- `409` — `slug` belongs to another college:
  ```json
  { "success": false, "error": "college with this slug already exists (slug=mit-college-of-engineering)" }
  ```

---

## Uploads endpoints

The API has three upload endpoints covering two storage backends:

- **Photos** (`logo`, `cover`, `gallery`) → **Cloudinary** (free tier — 25 monthly credits, 25 GB storage). The browser uploads directly to Cloudinary; the backend signs the request but never sees the bytes.
- **Documents** (`document` — PDFs only) → **Supabase Storage** (free tier — 1 GB). The backend streams the file from the client and returns a public URL.

This split keeps Cloudinary's image-optimization features (transformations, format negotiation) where they're useful and uses a dedicated doc store where they aren't. Photo endpoints require an authenticated admin or representative; the document endpoint is public so anonymous claim submitters can upload verification PDFs without first creating an account.

### End-to-end flow (photo upload — logo/cover/gallery)

```
┌────────────┐  1. POST /api/v1/uploads/sign       ┌──────────────┐
│            │  ─────────────────────────────────▶  │              │
│  Browser   │     {purpose: "logo"}                │  FMU backend │
│            │  ◀─────────────────────────────────  │              │
│            │     {signature, api_key,             └──────────────┘
│            │      timestamp, cloud_name,                       │
│            │      folder}                                      ▼
│            │                                          ┌──────────────┐
│            │  2. POST api.cloudinary.com/v1_1/...    │              │
│            │     multipart/form-data:               │  Cloudinary  │
│            │     file, api_key, timestamp,          │  (storage)   │
│            │     signature, folder                  │              │
│            │  ◀────────────────────────────────────  │              │
│            │     {secure_url, public_id, ...}        └──────────────┘
│            │                                                     │
│            │  3. POST /api/v1/universities                       │
│            │     { ..., logo: secure_url, cover_image: ... }     │
└────────────┘                                                     │
```

Each purpose (`logo`, `cover`, `gallery`) maps to a Cloudinary folder: `fmu/{APP_ENV}/{purpose}/`. Files auto-receive a unique filename on Cloudinary's side; overwrite is disabled. Documents use a different flow — see [`POST /api/v1/uploads/document`](#post-apiv1uploadsdocument) below.

### POST `/api/v1/uploads/sign`

Returns a one-shot signed-upload payload. Send it (along with the file) directly to Cloudinary — the backend never proxies the bytes.

**Auth:** admin or representative.

**Request body:**
```json
{ "purpose": "logo" }
```

`purpose` must be one of `logo`, `cover`, `gallery`, `document`. The response's `resource_type` field tells the frontend which Cloudinary endpoint to POST the file to — `image` for branding assets, `raw` for documents. Uploading a PDF to the `image` endpoint results in a `/image/upload/...pdf` URL that returns 401 on access; always use the `resource_type` from the response.

**Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "cloud_name": "your-cloud-name",
    "api_key": "123456789012345",
    "timestamp": 1715000000,
    "signature": "abc123...",
    "folder": "fmu/development/document",
    "resource_type": "raw"
  }
}
```

**Then from the browser, POST multipart to:**
```
POST https://api.cloudinary.com/v1_1/{cloud_name}/{resource_type}/upload
Content-Type: multipart/form-data

file=@<binary>          # the file
api_key=<api_key>       # from /sign response
timestamp=<timestamp>   # from /sign response
signature=<signature>   # from /sign response
folder=<folder>         # from /sign response
```

Cloudinary responds with:
```json
{
  "secure_url": "https://res.cloudinary.com/.../image/upload/v1234/fmu/development/logo/abc.jpg",
  "public_id": "fmu/development/logo/abc",
  "width": 800, "height": 800, "format": "jpg", "bytes": 12345,
  ...
}
```

Pass `secure_url` into the corresponding field on `POST /api/v1/universities`.

> Delivery-side transforms (`f_auto`, `q_auto`) should be applied via URL, not signed `eager` — eager transformations count against the 25 credits/month free tier.

**Errors:**
- `400` — `purpose` missing or not in allowlist
- `401` — missing/invalid session cookie
- `403` — role is not `admin`
- `500` — Cloudinary client not configured (missing `CLOUDINARY_*` env vars in non-dev)

### POST `/api/v1/uploads/image`

Optional server-side fallback for clients that cannot reach Cloudinary directly. Streams a multipart `file` to Cloudinary from the backend.

**Auth:** admin or representative.

**Query parameters:**
- `purpose` (required) — one of `logo`, `cover`, `gallery`

**Request body:** `multipart/form-data` with a single `file` field. Max 10 MiB. Allowlisted MIMEs: `image/jpeg`, `image/png`, `image/webp`, `image/gif`. Mime is detected from the first 512 bytes (form headers are not trusted).

**Response:** `201 Created`
```json
{
  "success": true,
  "data": {
    "secure_url": "https://res.cloudinary.com/...",
    "url": "http://res.cloudinary.com/...",
    "public_id": "fmu/development/logo/abc",
    "width": 800,
    "height": 800,
    "format": "jpg",
    "bytes": 12345
  }
}
```

**Errors:**
- `400` — missing/invalid `purpose`, or no `file` field
- `413` — file exceeds 10 MiB
- `415` — mime type not in allowlist
- `401` / `403` — requires admin or representative
- `502` — Cloudinary rejected the upload

### Storage / deletion

Cloudinary assets are referenced by URL in the database, not by FK. There is no automatic cleanup yet — when an `Update /universities/{id}` endpoint lands, replacing a logo/cover will best-effort `destroy` the previous Cloudinary public id in a goroutine (log-and-continue; never block the DB write on a Cloudinary outage).

### POST `/api/v1/uploads/document`

Upload a verification document (PDF only) to Supabase Storage. Returns a public URL that the frontend submits as `document_url` when [filing a university claim](#submit-a-claim). Unlike photo uploads, the file streams **through** the backend — there's no signed-redirect flow because documents are uploaded by anonymous users who don't have credentials yet.

**Auth:** public. The endpoint is intentionally unauthenticated so anyone can submit a claim without first registering.

**Query parameters:**
- `purpose` (required) — must be `document`. Any other value returns `400`.

**Request body:** `multipart/form-data` with a single `file` field. **Max 20 MiB.** Allowlisted MIME: `application/pdf` (detected from the first 512 bytes of the file — form headers are not trusted).

**Response:** `201 Created`
```json
{
  "success": true,
  "data": {
    "secure_url": "https://abcdefghijk.supabase.co/storage/v1/object/public/documents/4f8b2c1d9e3a7f6b5c0d2e8a1b9c4d7e.pdf",
    "path": "documents/4f8b2c1d9e3a7f6b5c0d2e8a1b9c4d7e.pdf",
    "bytes": 184320
  }
}
```

| Field        | Type   | Notes                                                                 |
|--------------|--------|-----------------------------------------------------------------------|
| `secure_url` | string | Public URL — paste this into the `document_url` field of the claim.   |
| `path`       | string | Storage key (`{bucket}/{filename}`). Reserved for future signed-URL flow; currently informational. |
| `bytes`      | int    | Size of the stored file in bytes.                                     |

**Object naming:** the server generates a random 32-char hex filename (e.g. `4f8b2c1d9e3a7f6b5c0d2e8a1b9c4d7e.pdf`). The bucket is public, but the filename is non-enumerable — guessing one is computationally infeasible.

**Errors:**
- `400` — `purpose` missing or not `document`
- `400` — no `file` field in the multipart body
- `400` — file is empty
- `413` — file exceeds 20 MiB
- `415` — mime type is not `application/pdf` (other MIME types are rejected before the file is forwarded to Supabase)
- `500` — Supabase client not configured (missing `SUPABASE_URL` / `SUPABASE_SERVICE_ROLE_KEY` env vars)
- `502` — Supabase rejected the upload (network error, auth failure, or bucket missing)

**Full claim submission example:**

```bash
# 1. Upload the PDF — get back a public URL
URL=$(curl -s -X POST "http://localhost:3000/api/v1/uploads/document?purpose=document" \
  -F "file=@/path/to/proof.pdf" | jq -r '.data.secure_url')

# 2. Submit the claim with that URL
curl -X POST "http://localhost:3000/api/v1/claims/universities/<university-uuid>" \
  -H "Content-Type: application/json" \
  -d "{
    \"full_name\": \"Ada Lovelace\",
    \"work_email\": \"ada@mit.edu\",
    \"document_url\": \"$URL\"
  }"
```

**Configuration:**

| Env var                     | Required | Default      | Notes                                                                 |
|-----------------------------|----------|--------------|-----------------------------------------------------------------------|
| `SUPABASE_URL`              | yes      | —            | Project URL, e.g. `https://abcdefghijk.supabase.co`                   |
| `SUPABASE_SERVICE_ROLE_KEY` | yes      | —            | **`service_role`** JWT from Project Settings → API. Server-only — never expose to the browser. |
| `SUPABASE_DOCS_BUCKET`      | no       | `documents`  | Bucket name. Must be set to **Public** in the Supabase dashboard.     |

Setup is one-time: create a Supabase project, create a public bucket named `documents`, copy the two keys into `.env`. The 20 MiB cap on the endpoint plus the 1 GB free-tier quota are the only abuse floors.

> **Going private later:** if verification PDFs ever need to be private, flip the bucket to Private in the Supabase dashboard and add a `GET /api/v1/admin/claims/{id}/document` endpoint that mints a short-lived signed URL on demand. The `path` field returned by this endpoint already carries everything needed (`{bucket}/{filename}`) so no schema change is required.

---

## Favorites endpoints

Students can save universities and colleges they like. All endpoints require
authentication **and** the `student` role — admins get `403`. The favorite
owner is always the authenticated user; there's no way to favorite on
someone else's behalf.

| Method | Path                                       | Description                                  |
|--------|--------------------------------------------|----------------------------------------------|
| POST   | `/api/v1/favorites/universities/{id}`      | Add a university to favorites (idempotent)   |
| DELETE | `/api/v1/favorites/universities/{id}`      | Remove a university from favorites (idempotent) |
| GET    | `/api/v1/favorites/universities`           | List the authenticated student's favorited universities (paginated) |
| POST   | `/api/v1/favorites/colleges/{id}`          | Add a college to favorites (idempotent)      |
| DELETE | `/api/v1/favorites/colleges/{id}`          | Remove a college from favorites (idempotent) |
| GET    | `/api/v1/favorites/colleges`               | List the authenticated student's favorited colleges (paginated) |

### Add to favorites
```bash
curl -b cookies.txt -X POST http://localhost:3000/api/v1/favorites/universities/<uuid>
```
Response (`200 OK`):
```json
{ "success": true, "data": null }
```

### Remove from favorites
```bash
curl -b cookies.txt -X DELETE http://localhost:3000/api/v1/favorites/universities/<uuid>
```
Response (`200 OK`):
```json
{ "success": true, "data": null }
```

### List favorites
```bash
curl -b cookies.txt http://localhost:3000/api/v1/favorites/universities
```
Response (`200 OK`):
```json
{
  "success": true,
  "data": {
    "items": [
      { "id": "...", "name": "MIT", "slug": "mit", "country": "USA", "..." }
    ],
    "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
  }
}
```

The list items are full [`UniversityListItem`](#university-list-item) /
[`CollegeListItem`](#college-list-item) objects — same shape as the public
list endpoints, so the frontend can render "My Favorites" without a second
request. Items are ordered by favorite-time, newest first. Supports
`?page=` and `?page_size=` (see [Pagination](#pagination)).

### Behavior notes

- **Idempotency:** `POST` and `DELETE` always return `200`, even if the row
  already existed or didn't exist. Repeating an add never duplicates a row.
- **Not found:** Favoriting a `uuid` that doesn't match any
  university/college returns `404 not found`. Deleting a non-existent
  favorite is silently a no-op.
- **Auth:** Missing cookie → `401 unauthorized`. Admin role → `403 forbidden`.

---

## Claim endpoints

Anyone can submit a claim to become the official representative of a university or college. Admins review both target types in one dashboard. Approving a claim creates a new `representative` user bound to that one target and returns a one-time plaintext password the admin must deliver securely.

**Public submission:**

| Method | Path                                      | Description |
|--------|-------------------------------------------|-------------|
| POST   | `/api/v1/claims/universities/{id}`        | Claim a university |
| POST   | `/api/v1/claims/colleges/{id}`            | Claim a college |

Both routes allow anonymous callers and authenticated callers whose role is neither `student` nor `admin`.

**Admin only** (require the `admin` role):

| Method | Path                                    | Description |
|--------|-----------------------------------------|-------------|
| GET    | `/api/v1/admin/claims`                  | List claims; filter by `type` and `status` |
| GET    | `/api/v1/admin/claims/{id}`             | Get a single university or college claim |
| POST   | `/api/v1/admin/claims/{id}/approve`     | Approve a claim and create its representative account |
| POST   | `/api/v1/admin/claims/{id}/reject`      | Reject a claim |

### Submit a claim

Choose the route from the target type; both accept the same JSON body:

- University: `POST /api/v1/claims/universities/{university_id}`
- College: `POST /api/v1/claims/colleges/{college_id}`

The `document_url` comes from a separate public `POST /api/v1/uploads/document?purpose=document` upload. That upload accepts PDF files up to 20 MiB and returns the URL as `data.secure_url`.

**Request body:**

```json
{
  "full_name": "Ada Lovelace",
  "work_email": "ada@mit.edu",
  "document_url": "https://abcdefghijk.supabase.co/storage/v1/object/public/documents/4f8b2c1d9e3a7f6b5c0d2e8a1b9c4d7e.pdf"
}
```

| Field | Type | Rules |
|-------|------|-------|
| `full_name` | string | required, 2–255 chars |
| `work_email` | string | required, valid email, max 255 chars |
| `document_url` | string | required, valid URL, max 500 chars |

`full_name` and `work_email` are trimmed before storage.

**College example:**

```bash
curl -X POST http://localhost:3000/api/v1/claims/colleges/9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10 \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Ada Lovelace",
    "work_email": "ada@mit.edu",
    "document_url": "https://abcdefghijk.supabase.co/storage/v1/object/public/documents/4f8b2c1d9e3a7f6b5c0d2e8a1b9c4d7e.pdf"
  }'
```

**Response:** `201 Created`

```json
{
  "success": true,
  "data": {
    "claim_id": "f8b6d642-8c31-4dc8-9ef2-195ba74a8b97",
    "type": "college",
    "target_id": "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10",
    "status": "pending",
    "created_at": "2026-07-23T10:55:59Z"
  }
}
```

The response uses the same shape for both routes. `type` is `"university"` or `"college"`; `target_id` is the claimed entity's ID.

**Errors:**

- `400` — malformed JSON or validation failure
- `403` — the request has a valid session for a `student` or `admin`:
  ```json
  { "success": false, "error": "only non-student, non-admin users may submit a claim" }
  ```
- `404` — the path ID does not identify an entity for the selected route:
  ```json
  { "success": false, "error": "target not found" }
  ```

### List claims (admin)

```bash
curl -b cookies.txt 'http://localhost:3000/api/v1/admin/claims?type=college&status=pending&page=1&page_size=20'
```

**Query params:**

| Param | Required | Values / behavior |
|-------|----------|-------------------|
| `type` | no | `university` or `college`; omit to include both tables |
| `status` | no | `pending`, `approved`, or `rejected`; omit for every status |
| `page` | no | default `1` |
| `page_size` | no | default `20`, max `100` |

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "f8b6d642-8c31-4dc8-9ef2-195ba74a8b97",
        "type": "college",
        "target_id": "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10",
        "target_name": "MIT College of Engineering",
        "full_name": "Ada Lovelace",
        "work_email": "ada@mit.edu",
        "document_url": "https://abcdefghijk.supabase.co/storage/v1/object/public/documents/4f8b2c1d9e3a7f6b5c0d2e8a1b9c4d7e.pdf",
        "status": "pending",
        "created_at": "2026-07-23T10:55:59Z",
        "updated_at": "2026-07-23T10:55:59Z"
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

Use `type` to choose the target icon, detail URL, and representative scope. `target_name` is joined from `universities` or `colleges`, so no extra request is needed for a moderation row.

This unified shape replaces the older university-specific `university_id` and `university_name` keys. Every claim item now uses:

```ts
type ClaimTarget = 'university' | 'college'

type ClaimItem = {
  id: string
  type: ClaimTarget
  target_id: string
  target_name: string
  full_name: string
  work_email: string
  document_url: string
  status: 'pending' | 'approved' | 'rejected'
  reviewer_id?: string
  reviewed_at?: string
  review_note?: string
  created_user_id?: string
  created_at: string
  updated_at: string
}
```

The review fields are omitted while they have no value; do not require them on pending rows.

For stable pagination, pass `type`. When `type` is omitted, the service queries each claim table with the same page settings and appends university results before college results. In that combined mode, `meta.total` is exact, but ordering and page boundaries are not global and `items` may contain up to twice `page_size`.

**Errors:**

- `400` with `type must be one of: university, college`
- `400` with `status must be one of: pending, approved, rejected`
- `401` — missing or invalid session
- `403` — authenticated user is not an admin

### Get one claim (admin)

```bash
curl -b cookies.txt http://localhost:3000/api/v1/admin/claims/f8b6d642-8c31-4dc8-9ef2-195ba74a8b97
```

**Response:** `200 OK` with one `ClaimItem` in `data`. The server finds the target type from the claim ID, so no `type` query parameter is needed.

**Errors:**

- `404` — `{ "success": false, "error": "claim not found" }`
- `401` — missing or invalid session
- `403` — authenticated user is not an admin

### Approve a claim (admin)

Approving creates a new `representative` user bound to the claim's `target_id`. The response contains the plaintext password **exactly once** — capture it immediately and deliver it securely to the representative.

The body must be a JSON object. `review_note` is optional and has a maximum length of 1,000 characters; send `{}` when there is no note.

```bash
curl -b cookies.txt -X POST http://localhost:3000/api/v1/admin/claims/f8b6d642-8c31-4dc8-9ef2-195ba74a8b97/approve \
  -H "Content-Type: application/json" \
  -d '{"review_note":"Verified employee via HR letter."}'
```

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "claim": {
      "id": "f8b6d642-8c31-4dc8-9ef2-195ba74a8b97",
      "type": "college",
      "target_id": "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10",
      "target_name": "MIT College of Engineering",
      "full_name": "Ada Lovelace",
      "work_email": "ada@mit.edu",
      "document_url": "https://abcdefghijk.supabase.co/storage/v1/object/public/documents/4f8b2c1d9e3a7f6b5c0d2e8a1b9c4d7e.pdf",
      "status": "approved",
      "reviewer_id": "<admin-uuid>",
      "reviewed_at": "2026-07-23T11:00:00Z",
      "review_note": "Verified employee via HR letter.",
      "created_user_id": "<new-rep-uuid>",
      "created_at": "2026-07-23T10:55:59Z",
      "updated_at": "2026-07-23T11:00:00Z"
    },
    "created_user_id": "<new-rep-uuid>",
    "email": "ada@mit.edu",
    "plain_password": "a1b2c3d4e5f6a7b8",
    "role": "representative"
  }
}
```

The new user receives `representative_university_id = target_id` for a university claim or `representative_college_id = target_id` for a college claim. The representative can log in immediately with `email` and `plain_password`.

After a successful approval the backend sends a credentials email to the address on the claim (the new representative's `email`). Delivery is best-effort — when SMTP is not configured the app falls back to a no-op sender and the API response is unchanged. Configure the `MAIL_*` environment variables (see `.env`) to enable real delivery; the email contains the same `email` and `plain_password` returned here.

**Errors:**

- `400` — malformed body or `review_note` exceeds 1,000 characters
- `404` — claim not found
- `409` — claim has already been reviewed
- `409` — university already has a representative
- `409` — college already has a representative
- `401` — missing or invalid session
- `403` — authenticated user is not an admin

### Reject a claim (admin)

The body follows the same rules as approval: send a JSON object, with an optional `review_note` up to 1,000 characters.

```bash
curl -b cookies.txt -X POST http://localhost:3000/api/v1/admin/claims/f8b6d642-8c31-4dc8-9ef2-195ba74a8b97/reject \
  -H "Content-Type: application/json" \
  -d '{"review_note":"Document did not verify employment."}'
```

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "claim": {
      "id": "f8b6d642-8c31-4dc8-9ef2-195ba74a8b97",
      "type": "college",
      "target_id": "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10",
      "target_name": "MIT College of Engineering",
      "full_name": "Ada Lovelace",
      "work_email": "ada@mit.edu",
      "document_url": "https://abcdefghijk.supabase.co/storage/v1/object/public/documents/4f8b2c1d9e3a7f6b5c0d2e8a1b9c4d7e.pdf",
      "status": "rejected",
      "reviewer_id": "<admin-uuid>",
      "reviewed_at": "2026-07-23T11:00:00Z",
      "review_note": "Document did not verify employment.",
      "created_at": "2026-07-23T10:55:59Z",
      "updated_at": "2026-07-23T11:00:00Z"
    }
  }
}
```

**Errors:**

- `400` — malformed body or `review_note` exceeds 1,000 characters
- `404` — claim not found
- `409` — claim has already been reviewed
- `401` — missing or invalid session
- `403` — authenticated user is not an admin

### Representative editing rights

After login, use `role` and the optional scope IDs from `/auth/login`, `/auth/refresh`, or `/auth/me` to render editing controls:

```ts
type SessionUser = {
  user_id: string
  full_name: string
  email: string
  avatar?: string
  role: 'student' | 'representative' | 'admin'
  representative_university_id?: string
  representative_college_id?: string
}
```

A representative account created from a claim has one scope ID, never both. The server copies that binding into the HttpOnly access token and enforces it on every write.

| Account scope | Endpoint | Enforcement |
|---------------|----------|-------------|
| University representative | `PATCH /api/v1/universities/{id}` | Path `{id}` must equal `representative_university_id`; `name` and `slug` are admin-only |
| University representative | `POST /api/v1/colleges` | Body `university_id` must equal `representative_university_id` |
| College representative | `PATCH /api/v1/colleges/{id}` | Path `{id}` must equal `representative_college_id`; `name` and `slug` are admin-only |
| Either representative scope | `POST /api/v1/uploads/sign` | Role-gated; the uploaded URL is authorized when saved through a scoped write |
| Either representative scope | `POST /api/v1/uploads/image` | Same as `/uploads/sign` |

Admins may use every endpoint in the table for any target. Scope is still checked server-side, so hiding a button in the UI is only a usability improvement, not an authorization control.

Public university and college detail, list, and search payloads include `has_representative`. When it is `true`, remove or disable the "claim this institution" banner; do not query the admin claims feed to infer ownership.

Important distinctions:

- A university representative may create colleges under their university but cannot patch those colleges unless separately bound as that college's representative.
- A college representative may patch their college but cannot patch its parent university or create another college.
- Representatives must omit `name` and `slug` from university and college PATCH bodies. Supplying either key returns `403`, even if its value is unchanged.
- Representatives cannot create universities, read university admin stats, review claims, or access any `/api/v1/admin/*` route.
- `POST /api/v1/uploads/document` remains public because it supports anonymous claim submission.

A path-scope mismatch returns:

```json
{ "success": false, "error": "forbidden" }
```

A university representative sending another university's ID in `POST /api/v1/colleges` returns:

```json
{ "success": false, "error": "representative can only edit their own university" }
```

---

## Counselling endpoints

Two flavours of free-counselling form, both fully public — no authentication required to submit:

1. **General counselling** — not tied to any institution. Carries a free-text `preferred_country`, an optional `preferred_university` name, and an optional resume upload.
2. **Institution-specific counselling** — submitted from a university or college detail page. Carries richer academic context (program of interest, start term, current education, test scores, free-form message). `phone` and `start_term` are required.

Both rows are visible to admins. Institution-specific rows are also visible to that institution's representative so they can follow up; representatives can also update status. General rows are admin-only because they have no natural owner.

### Resume upload

Before submitting a general counselling request with a resume, the frontend uploads the file and captures `secure_url`:

### POST `/api/v1/uploads/resume`

Upload a CV/resume to Supabase Storage. Accepts **PDF, DOC, DOCX** (max **5 MiB**). Filename extension is what gates acceptance — `http.DetectContentType` misclassifies DOC/DOCX, so the handler sniffs by extension. The returned `secure_url` is what the counselling form submits as `resume_url`.

**All auth:** public

**Request:**
```bash
curl -X POST http://localhost:3000/api/v1/uploads/resume \
  -F "file=@/path/to/cv.pdf"
```

**Response 201:**
```json
{
  "success": true,
  "data": {
    "secure_url": "https://<project>.supabase.co/storage/v1/object/public/documents/<hex>.pdf",
    "path": "documents/<hex>.pdf",
    "bytes": 102400
  }
}
```

**Errors:**
- `400` — invalid multipart form, missing `file`, empty file
- `413` — file exceeds 5 MiB
- `415` — file extension is not `.pdf`, `.doc`, or `.docx`
- `502` — Supabase upload failed

### Submit a general counselling request

### POST `/api/v1/counselling/general`

Public submit endpoint for the home-page "Free Counselling" form.

**All auth:** public

**Request body:**
```json
{
  "full_name": "John Smith",
  "email": "john@example.com",
  "phone": "+1 234 567 8900",
  "country": "United States",
  "preferred_university": "Harvard, MIT, ... (optional)",
  "resume_url": "https://<project>.supabase.co/.../cv.pdf"
}
```

| Field | Required | Notes |
|-------|----------|-------|
| `full_name` | yes | 2–255 chars |
| `email` | yes | valid email |
| `phone` | no | ≤ 50 chars |
| `country` | yes | 2–100 chars — preferred study country |
| `preferred_university` | no | ≤ 255 chars |
| `resume_url` | no | URL returned by `/uploads/resume` |

**Response 201:**
```json
{
  "success": true,
  "data": {
    "inquiry_id": "uuid",
    "type": "general",
    "status": "pending",
    "created_at": "2026-07-25T10:00:00Z"
  }
}
```

### Submit an institution-specific counselling request

### POST `/api/v1/counselling/universities/{id}`

### POST `/api/v1/counselling/colleges/{id}`

Public submit endpoints for the "Request Free Counselling" form on university and college detail pages. The `{id}` path segment is the target institution's UUID; the response carries it back in `target_id`.

**All auth:** public

**Request body:**
```json
{
  "full_name": "John Smith",
  "email": "john@example.com",
  "phone": "+1 (555) 000-0000",
  "country": "United States",
  "program_of_interest": "Computer Science",
  "start_term": "Fall 2027",
  "current_education": "High School Senior",
  "test_scores": "SAT: 1400, ACT: 32",
  "message": "Looking for scholarships in CS..."
}
```

| Field | Required | Notes |
|-------|----------|-------|
| `full_name` | yes | 2–255 chars |
| `email` | yes | valid email |
| `phone` | yes | 4–50 chars |
| `country` | no | ≤ 100 chars — applicant's country |
| `program_of_interest` | no | ≤ 255 chars |
| `start_term` | yes | 1–100 chars |
| `current_education` | no | ≤ 100 chars |
| `test_scores` | no | ≤ 1000 chars |
| `message` | no | ≤ 5000 chars |

**Response 201:**
```json
{
  "success": true,
  "data": {
    "inquiry_id": "uuid",
    "type": "university",
    "target_id": "uuid-of-the-target-institution",
    "status": "pending",
    "created_at": "2026-07-25T10:00:00Z"
  }
}
```

**Errors:**
- `400` — missing required field (e.g. `phone` or `start_term`)
- `404` — `{id}` does not match a university / college

### Status workflow

`pending` (default) → `reviewed` → `archived`. The PATCH endpoint accepts only `reviewed` or `archived`; once touched, a row cannot go back to `pending`. `reviewed_at` is stamped on the first transition to `reviewed` and stays set thereafter. `reviewer_id` is stamped to the calling user.

### List counselling inquiries (admin)

### GET `/api/v1/admin/counselling`

Paginated list. Admins see everything; `?type=` and `?status=` narrow the feed.

**Auth:** admin

**Query parameters:**

| Param | Type | Default | Notes |
|-------|------|---------|-------|
| `type` | enum | (all) | `general`, `university`, `college` |
| `status` | enum | (all) | `pending`, `reviewed`, `archived` |
| `page` | int | 1 | |
| `page_size` | int | 20 | max 100 |

**Response 200:**
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "uuid",
        "type": "university",
        "target_id": "uuid",
        "target_name": "Stanford University",
        "full_name": "John Smith",
        "email": "john@example.com",
        "phone": "+1 555 000 0000",
        "country": "United States",
        "program_of_interest": "Computer Science",
        "start_term": "Fall 2027",
        "current_education": "High School Senior",
        "test_scores": "SAT: 1400",
        "message": "...",
        "status": "pending",
        "created_at": "2026-07-25T10:00:00Z",
        "updated_at": "2026-07-25T10:00:00Z"
      }
    ],
    "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
  }
}
```

General rows omit `target_id` / `target_name` and never carry `phone`/`program_of_interest`/etc.; university/college rows omit `preferred_university` / `resume_url`.

### Get one counselling inquiry (admin)

### GET `/api/v1/admin/counselling/{id}`

**Auth:** admin

**Response 200:** the same row shape as the list endpoint, returned alone under `data`.

### Update a counselling inquiry (admin)

### PATCH `/api/v1/admin/counselling/{id}`

Transition status to `reviewed` or `archived`, optionally attach a private `review_note`.

**Auth:** admin

**Request body:**
```json
{ "status": "reviewed", "review_note": "Contacted via email on 2026-07-26" }
```

**Response 200:** the updated row (with `reviewer_id`, `reviewed_at`, and `updated_at` refreshed).

### Representative counselling endpoints

The same three operations exist under `/representative/counselling`. Representatives only see rows bound to their institution (their JWT carries `representative_university_id` or `representative_college_id`). General rows are invisible to representatives — `GET /representative/counselling?type=general` returns an empty page. Any out-of-scope read or write returns `403`.

### GET `/api/v1/representative/counselling?status=pending&page=1`

### GET `/api/v1/representative/counselling/{id}`

### PATCH `/api/v1/representative/counselling/{id}`

Same request/response shapes as the admin counterparts; same auth role gating (`admin` or `representative`), but with representative scope enforced server-side.

### Frontend integration checklist

- [ ] Home-page "Free Counselling" form: POST the body to `/counselling/general`. If the user attached a resume, upload first via `/uploads/resume` (≤ 5 MiB, PDF/DOC/DOCX) and submit the returned `secure_url` as `resume_url`.
- [ ] University/college detail page "Request Free Counselling" form: POST the body to `/counselling/universities/{id}` or `/counselling/colleges/{id}`. `phone` and `start_term` are required.
- [ ] After submit: render a success state and reset the form. Do not poll for status — the user is notified by email out-of-band.
- [ ] Admin counselling dashboard: list via `/admin/counselling?type=university&status=pending` (separate calls per type for deterministic pagination). Filter by `status` to focus on the active queue.
- [ ] Admin/rep update: PATCH `{status: "reviewed", review_note: "..."}` to mark handled. `status: "archived"` removes a row from the default `pending` feed without deleting it.
- [ ] Representative dashboard: list via `/representative/counselling?status=pending` — the service scopes this automatically to the representative's institution.
- [ ] Hide the resume field for institution-specific forms; the columns don't apply there. The backend ignores `resume_url` on specific rows.

---
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

- [ ] On app load: call `GET /api/v1/auth/me` to bootstrap the current user (treat a 401 as "not logged in"). Fetch `/api/v1/universities/lookups` and cache.
- [ ] For cross-origin SPAs, configure your HTTP client to send `credentials: 'include'` on every request (same-origin requests get this for free)
- [ ] On 401 from any protected endpoint: call `POST /api/v1/auth/refresh` (no body) and retry the original request; on second 401, clear the bootstrapped user state and redirect to login
- [ ] On logout: `DELETE /api/v1/auth/logout` (no body) — the server clears the cookies; the SPA can also clear any cached user state
- [ ] University list page: `GET /universities?page=N&page_size=20&<filters>`, render items + meta. See "Filters" in the List endpoint above — filter param names match the FilterSidebar's `searchParams` keys exactly. Use each row's `has_representative` to hide the "claim this university" prompt when it is `true`.
- [ ] University detail page: `GET /universities/{id}`, render all fields + lookup arrays. Hide the representative-claim banner when `has_representative` is `true` (anonymous viewers included).
- [ ] University detail page — affiliated colleges list: `GET /universities/{id}/colleges` (or `GET /colleges?university_id={id}` with filters). Each college row carries `has_representative` for the same banner logic.
- [ ] College list page: `GET /colleges?page=N&page_size=20&<filters>` (`university_id`, `country`, `state_province`, `city`). Use `has_representative` on each row to hide the claim banner.
- [ ] College search box: `GET /colleges/search?q=<term>` — debounced; one search box covers college name, college location, and parent university name/slug. Results embed the parent university and `is_favorited`/`has_representative`, so a single render call is enough.
- [ ] College detail page: `GET /colleges/{id}`, render all fields including the new profile metadata (`excerpt`, `contact_*`, `website`, `cover_image`, `gallery_images`, `institution_type`, `campus_setting`, `founded_year`, `campus_size`, `is_popular`, `is_featured`). Hide the claim banner when `has_representative` is `true`.
- [ ] Admin "create university" form: 
  - Use cached lookups to populate multi-selects
  - On submit: `POST /universities` with the assembled payload
  - On 400: show the `errors[]` list inline next to each field
  - On 201: redirect to the new detail page
- [ ] Admin "create college" form: `POST /colleges` with `university_id` selected from the universities list
- [ ] Admin promotion: not a frontend concern; backend operator runs the SQL
- [ ] "Become a representative" claim form (public):
  - Determine whether the user is claiming a university or college and retain that target's ID
  - Upload the verification PDF first: `POST /uploads/document?purpose=document` (multipart, ≤ 20 MiB, PDF only; no auth required)
  - Submit `{full_name, work_email, document_url}` to `/claims/universities/{id}` or `/claims/colleges/{id}`
  - Read the unified response as `{claim_id, type, target_id, status, created_at}`
- [ ] Admin claims dashboard:
  - Prefer `GET /admin/claims?type=university&status=pending&page=N` and a separate `type=college` view for deterministic pagination
  - Render each row from `type`, `target_id`, and `target_name`; do not use the removed `university_id` / `university_name` keys
  - Approve with `POST /admin/claims/{id}/approve`; capture and display the returned `plain_password` exactly once
  - Reject with `POST /admin/claims/{id}/reject`
- [ ] Representative editing UI:
  - Bootstrap with `/auth/me` and branch on `representative_university_id` versus `representative_college_id`
  - University rep: show edit controls only for `/universities/{representative_university_id}` and allow college creation beneath it. Disable the `name` and `slug` fields in the form (representatives get `403` if they try to PATCH them)
  - College rep: show edit controls only for `/colleges/{representative_college_id}` and save through `PATCH /colleges/{id}`. Disable the `name` and `slug` fields in the form (representatives get `403` if they try to PATCH them)
  - Treat backend `403` responses as authoritative even if the UI believed the user was in scope
  - Public read payloads already expose `has_representative`, so use it to hide the claim banner — no need to query the admin claims feed to detect ownership

---

## CORS

CORS is configured via the `ALLOWED_ORIGINS` env var (comma-separated list of origins, e.g. `ALLOWED_ORIGINS=http://localhost:3001`). Defaults to empty, which blocks all cross-origin requests. The middleware is mounted at the top of the chi router in `cmd/api/main.go` (`github.com/go-chi/cors`), so preflight `OPTIONS` requests are handled before any auth middleware runs.

`AllowCredentials: true` is enabled, which is required for the auth cookies to flow on cross-origin requests. As a consequence, the origin list **cannot include `*`** — the server refuses to start if it does. List explicit origins (`http://localhost:3001,https://app.example.com`).

The SPA must opt in by sending `credentials: 'include'` (or `withCredentials: true` on `XMLHttpRequest`) on every fetch — same-origin requests get this for free, cross-origin requests do not.