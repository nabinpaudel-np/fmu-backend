-- name: CreateProgram :one
INSERT INTO programs (title, description, excerpt, career_options, degree_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, title, description, degree_id, created_at, updated_at, excerpt, career_options;

-- name: GetProgramByID :one
SELECT id, title, description, degree_id, created_at, updated_at, excerpt, career_options
FROM programs
WHERE id = $1;

-- name: DeleteProgram :execrows
DELETE FROM programs WHERE id = $1;

-- name: ListPrograms :many
SELECT id, title, description, degree_id, created_at, updated_at, excerpt, career_options
FROM programs
ORDER BY title
LIMIT $1 OFFSET $2;

-- name: CountPrograms :one
SELECT COUNT(*) FROM programs;

-- name: ListProgramsByDegree :many
SELECT id, title, description, degree_id, created_at, updated_at, excerpt, career_options
FROM programs
WHERE degree_id = $1
ORDER BY title
LIMIT $2 OFFSET $3;

-- name: CountProgramsByDegree :one
SELECT COUNT(*) FROM programs WHERE degree_id = $1;

-- name: ListAllPrograms :many
SELECT id, title, description, degree_id, created_at, updated_at, excerpt, career_options
FROM programs
ORDER BY title;

-- name: UpdateProgram :one
UPDATE programs
SET title = $2,
    description = $3,
    excerpt = $4,
    career_options = $5,
    degree_id = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING id, title, description, degree_id, created_at, updated_at, excerpt, career_options;
