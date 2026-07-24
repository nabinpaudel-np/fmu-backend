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