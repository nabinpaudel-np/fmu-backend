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