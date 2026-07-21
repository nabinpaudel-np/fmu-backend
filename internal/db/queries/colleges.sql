-- name: CreateCollege :one
INSERT INTO colleges (name, slug, university_id, overview, country, state, city, full_location, logo) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING *;

-- name: GetCollegeByID :one
SELECT * FROM colleges WHERE id = $1;

-- name: ListCollegesByUniversity :many
SELECT * FROM colleges WHERE university_id = $1 ORDER BY name LIMIT $2 OFFSET $3;

-- name: CountCollegesByUniversity :one
SELECT COUNT(*) FROM colleges WHERE university_id = $1;

-- name: SearchColleges :many
-- Typo-tolerant search across college + parent-university fields via pg_trgm.
-- JOIN is INNER because college.university_id has ON DELETE RESTRICT and is
-- NOT NULL — every college has exactly one parent university. SELECT re-aliases
-- `u.id AS university_id` so the API response carries the university id once
-- (it equals college.university_id, but reading u.id keeps the join honest).
-- COALESCE keeps nullable columns as plain strings in the row type.
SELECT
    c.id,
    c.name,
    c.slug,
    u.id AS university_id,
    u.name AS university_name,
    u.slug AS university_slug,
    COALESCE(u.logo, '') AS university_logo,
    COALESCE(c.country, '') AS country,
    COALESCE(c.state, '') AS state,
    COALESCE(c.city, '') AS city,
    COALESCE(c.full_location, '') AS full_location,
    COALESCE(c.logo, '') AS logo
FROM colleges c
JOIN universities u ON u.id = c.university_id
WHERE similarity(c.name, $1) > 0.2
   OR similarity(c.full_location, $1) > 0.2
   OR similarity(c.city, $1) > 0.2
   OR similarity(c.state, $1) > 0.2
   OR similarity(c.country, $1) > 0.2
   OR similarity(u.name, $1) > 0.2
   OR similarity(u.slug, $1) > 0.2
ORDER BY GREATEST(
    similarity(c.name, $1),
    similarity(c.full_location, $1),
    similarity(c.city, $1),
    similarity(u.name, $1),
    similarity(u.slug, $1)
) DESC, c.name ASC
LIMIT $2;
