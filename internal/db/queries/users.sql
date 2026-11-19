-- name: CreateUser :one
INSERT INTO users (full_name, email, password, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByProvider :one
SELECT * FROM users
WHERE oauth_provider = $1 AND oauth_id = $2;

-- name: CreateUserWithOAuth :one
INSERT INTO users (full_name, email, oauth_provider, oauth_id, avatar, email_verified, role)
VALUES ($1, $2, $3, $4, $5, true, 'student')
RETURNING *;

-- name: CreateRepresentativeUser :one
INSERT INTO users (full_name, email, password, role, representative_university_id, email_verified)
VALUES ($1, $2, $3, 'representative', $4, true)
RETURNING *;

-- name: CreateCollegeRepresentativeUser :one
INSERT INTO users (full_name, email, password, role, representative_college_id, email_verified)
VALUES ($1, $2, $3, 'representative', $4, true)
RETURNING *;

-- name: CountUsers :one
-- Counts the users the admin list would show, honouring the same filters.
SELECT COUNT(*) FROM users
WHERE ($1::text = '' OR role = $1)
  AND ($2::text = '' OR full_name ILIKE '%' || $2 || '%' OR email ILIKE '%' || $2 || '%');

-- name: ListUsers :many
-- The admin user list. Filters are optional strings: an empty role matches
-- every role, an empty search matches every name and email. The search is
-- matched case-insensitively so an admin can type part of an email address.
--
-- The institution a representative is bound to is joined in by name so the
-- dashboard can say which one it is instead of showing a bare UUID. The
-- joins are LEFT because the institution may have been deleted while the
-- binding still stands — an inner join would drop that user from the list
-- entirely.
SELECT
    u.id, u.full_name, u.avatar, u.email, u.password, u.oauth_provider, u.oauth_id,
    u.email_verified, u.created_at, u.updated_at, u.role,
    u.representative_university_id, u.representative_college_id,
    uni.name AS representative_university_name,
    col.name AS representative_college_name
FROM users u
LEFT JOIN universities uni ON uni.id = u.representative_university_id
LEFT JOIN colleges col ON col.id = u.representative_college_id
WHERE ($1::text = '' OR u.role = $1)
  AND ($2::text = '' OR u.full_name ILIKE '%' || $2 || '%' OR u.email ILIKE '%' || $2 || '%')
ORDER BY u.created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetAdminUserByID :one
-- One user for the admin detail endpoint, with the name of the institution
-- a representative is bound to. Same shape as the list row.
SELECT
    u.id, u.full_name, u.avatar, u.email, u.password, u.oauth_provider, u.oauth_id,
    u.email_verified, u.created_at, u.updated_at, u.role,
    u.representative_university_id, u.representative_college_id,
    uni.name AS representative_university_name,
    col.name AS representative_college_name
FROM users u
LEFT JOIN universities uni ON uni.id = u.representative_university_id
LEFT JOIN colleges col ON col.id = u.representative_college_id
WHERE u.id = $1;

-- name: CountAdmins :one
-- How many admins exist. Used to refuse the removal of the last one.
SELECT COUNT(*) FROM users WHERE role = 'admin';

-- name: UpdateUserRole :one
-- Changes a user's role and the institution they are bound to.
--
-- The scope columns are cleared when the new role is not "representative",
-- because a representative is bound to exactly one university or college and
-- a stale binding would keep granting edit rights over it. The UNIQUE
-- constraints on those columns would also stop another representative from
-- ever claiming the same institution.
--
-- The scope is given as a string. An empty string means "clear this binding",
-- and NULLIF turns that into a real NULL — a blank uuid is not a valid one
-- and the foreign key would reject it.
--
-- The institution names are joined in for the same reason as in the list, so
-- a role change returns the same shape and the dashboard can update the row
-- in place without a second request.
UPDATE users u
SET role = $2,
    representative_university_id = NULLIF($3, '')::uuid,
    representative_college_id = NULLIF($4, '')::uuid,
    updated_at = now()
WHERE u.id = $1
RETURNING
    u.id, u.full_name, u.avatar, u.email, u.password, u.oauth_provider, u.oauth_id,
    u.email_verified, u.created_at, u.updated_at, u.role,
    u.representative_university_id, u.representative_college_id,
    (COALESCE((SELECT uni.name FROM universities uni WHERE uni.id = NULLIF($3, '')::uuid), ''))::text AS representative_university_name,
    (COALESCE((SELECT col.name FROM colleges col WHERE col.id = NULLIF($4, '')::uuid), ''))::text AS representative_college_name;
-- name: DeleteUser :exec
-- Removes the user. The rows that reference them are removed with them:
-- refresh_tokens, favorites and the claims they created all cascade.
DELETE FROM users WHERE id = $1;

-- name: InvalidateUnusedPasswordResetTokens :exec
-- Burns every unused reset token for a user so that only the newest link
-- works. An admin-triggered reset makes any earlier link useless.
UPDATE password_reset_tokens
SET used_at = now()
WHERE user_id = $1 AND used_at IS NULL;