-- name: CreateCollegeClaim :one
INSERT INTO college_claims (college_id, full_name, work_email, role, document_url)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetCollegeClaimByID :one
-- JOINs colleges so the admin dashboard can show the college name
-- alongside the claim row in a single round-trip.
SELECT
    c.id, c.college_id, co.name AS college_name,
    c.full_name, c.work_email, c.role, c.document_url,
    c.status, c.reviewer_id, c.reviewed_at, c.review_note,
    c.created_user_id, c.created_at, c.updated_at
FROM college_claims c
JOIN colleges co ON co.id = c.college_id
WHERE c.id = $1;

-- name: CountCollegeClaims :one
SELECT COUNT(*) FROM college_claims
WHERE ($1::text IS NULL OR status = $1);

-- name: ListCollegeClaims :many
SELECT
    c.id, c.college_id, co.name AS college_name,
    c.full_name, c.work_email, c.role, c.document_url,
    c.status, c.reviewer_id, c.reviewed_at, c.review_note,
    c.created_user_id, c.created_at, c.updated_at
FROM college_claims c
JOIN colleges co ON co.id = c.college_id
WHERE ($1::text IS NULL OR c.status = $1)
ORDER BY c.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPendingClaimsForCollege :one
SELECT COUNT(*) FROM college_claims
WHERE college_id = $1 AND status = 'pending';

-- name: ApproveCollegeClaim :one
UPDATE college_claims
SET status = 'approved',
    reviewer_id = $2,
    reviewed_at = NOW(),
    review_note = $3,
    created_user_id = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: RejectCollegeClaim :one
UPDATE college_claims
SET status = 'rejected',
    reviewer_id = $2,
    reviewed_at = NOW(),
    review_note = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;