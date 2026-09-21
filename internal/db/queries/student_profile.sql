-- name: UpsertStudentProfile :one
INSERT INTO student_profile (user_id, budget, intended_country, current_education, updated_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (user_id) DO UPDATE
   SET budget           = COALESCE(EXCLUDED.budget,           student_profile.budget),
       intended_country  = COALESCE(EXCLUDED.intended_country,  student_profile.intended_country),
       current_education = COALESCE(EXCLUDED.current_education, student_profile.current_education),
       updated_at = NOW()
RETURNING *;

-- name: GetStudentProfile :one
SELECT * FROM student_profile WHERE user_id = $1;

-- name: DeleteStudentProfile :exec
DELETE FROM student_profile WHERE user_id = $1;

-- name: DeleteStudentProfilePrograms :exec
DELETE FROM student_profile_programs WHERE user_id = $1;

-- name: AddStudentProfileProgram :exec
INSERT INTO student_profile_programs (user_id, program_id) VALUES ($1, $2)
ON CONFLICT (user_id, program_id) DO NOTHING;

-- name: ListStudentProfileProgramIDs :many
SELECT program_id FROM student_profile_programs WHERE user_id = $1;

-- name: GetExistingProgramIDs :many
SELECT id FROM programs WHERE id = ANY($1::uuid[]);