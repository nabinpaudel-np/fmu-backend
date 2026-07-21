-- name: CreateUniversityClaim :one
INSERT INTO university_claims (university_id, full_name, work_email, document_url)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUniversityClaimByID :one
-- JOINs universities so the admin dashboard can show the university name
-- alongside the claim row in a single round-trip.
SELECT
    c.id, c.university_id, u.name AS university_name,
    c.full_name, c.work_email, c.document_url,
    c.status, c.reviewer_id, c.reviewed_at, c.review_note,
    c.created_user_id, c.created_at, c.updated_at
FROM university_claims c
JOIN universities u ON u.id = c.university_id
WHERE c.id = $1;

-- name: CountUniversityClaims :one
SELECT COUNT(*) FROM university_claims
WHERE ($1::text IS NULL OR status = $1);

-- name: ListUniversityClaims :many
SELECT
    c.id, c.university_id, u.name AS university_name,
    c.full_name, c.work_email, c.document_url,
    c.status, c.reviewer_id, c.reviewed_at, c.review_note,
    c.created_user_id, c.created_at, c.updated_at
FROM university_claims c
JOIN universities u ON u.id = c.university_id
WHERE ($1::text IS NULL OR c.status = $1)
ORDER BY c.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPendingClaimsForUniversity :one
SELECT COUNT(*) FROM university_claims
WHERE university_id = $1 AND status = 'pending';

-- name: CountActiveRepresentativeForUniversity :one
SELECT COUNT(*) FROM users
WHERE representative_university_id = $1;

-- name: ApproveUniversityClaim :one
UPDATE university_claims
SET status = 'approved',
    reviewer_id = $2,
    reviewed_at = NOW(),
    review_note = $3,
    created_user_id = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: RejectUniversityClaim :one
UPDATE university_claims
SET status = 'rejected',
    reviewer_id = $2,
    reviewed_at = NOW(),
    review_note = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;
