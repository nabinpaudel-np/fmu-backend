-- name: CreateProgram :one
INSERT INTO programs (title, description, degree_id)
VALUES ($1, $2, $3)
RETURNING id, title, description, degree_id, created_at, updated_at;

-- name: GetProgramByID :one
SELECT id, title, description, degree_id, created_at, updated_at
FROM programs
WHERE id = $1;

-- name: DeleteProgram :execrows
DELETE FROM programs WHERE id = $1;

-- name: ListPrograms :many
SELECT id, title, description, degree_id, created_at, updated_at
FROM programs
ORDER BY title
LIMIT $1 OFFSET $2;

-- name: CountPrograms :one
SELECT COUNT(*) FROM programs;

-- name: ListProgramsByDegree :many
SELECT id, title, description, degree_id, created_at, updated_at
FROM programs
WHERE degree_id = $1
ORDER BY title
LIMIT $2 OFFSET $3;

-- name: CountProgramsByDegree :one
SELECT COUNT(*) FROM programs WHERE degree_id = $1;

-- name: ListAllPrograms :many
SELECT id, title, description, degree_id, created_at, updated_at
FROM programs
ORDER BY title;
