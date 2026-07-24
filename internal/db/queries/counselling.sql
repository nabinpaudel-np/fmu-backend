-- name: CreateCounsellingInquiry :one
INSERT INTO counselling_inquiries (
    target_type, target_id, full_name, email, phone, country,
    preferred_university, program_of_interest, start_term,
    current_education, test_scores, message, resume_url
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9,
    $10, $11, $12, $13
)
RETURNING *;

-- name: GetCounsellingInquiryByID :one
-- NULL-safe LEFT JOINs produce a single row regardless of which (if any)
-- target table the row points at. target_name is populated only when
-- target_type is set; the service maps NULL into an empty string.
SELECT
    ci.id, ci.target_type, ci.target_id, ci.full_name, ci.email,
    ci.phone, ci.country, ci.preferred_university,
    ci.program_of_interest, ci.start_term, ci.current_education,
    ci.test_scores, ci.message, ci.resume_url,
    ci.status, ci.reviewer_id, ci.reviewed_at, ci.review_note,
    ci.created_at, ci.updated_at,
    COALESCE(u.name, co.name, '') AS target_name
FROM counselling_inquiries ci
LEFT JOIN universities u
    ON ci.target_type = 'university' AND u.id = ci.target_id
LEFT JOIN colleges co
    ON ci.target_type = 'college' AND co.id = ci.target_id
WHERE ci.id = $1;

-- name: CountCounsellingInquiries :one
-- Filter contract for the target dimension ($2, $3):
--   $2 NULL or ''                        → no target filter, match anything
--   $2 = '__general__'                   → only general rows (target_type IS NULL)
--   $2 = 'university' | 'college'        → match that target_type, optionally
--                                          narrowed by target_id ($3)
--
-- We use NULLIF on $2 and $3 because sqlc hands us string params and Go's
-- empty string ("") is not the same as SQL NULL — without NULLIF the "no
-- filter" case would compare target_type to '' and match nothing.
SELECT COUNT(*) FROM counselling_inquiries
WHERE (NULLIF($1, '') IS NULL OR status = $1)
  AND (
        (NULLIF($2, '') IS NULL)
     OR ($2 = '__general__' AND target_type IS NULL)
     OR ($2 <> '__general__' AND target_type = $2
            AND (NULLIF($3, '') IS NULL OR target_id::text = $3))
  );

-- name: ListCounsellingInquiries :many
-- Same filter contract as CountCounsellingInquiries; see comment there.
SELECT
    ci.id, ci.target_type, ci.target_id, ci.full_name, ci.email,
    ci.phone, ci.country, ci.preferred_university,
    ci.program_of_interest, ci.start_term, ci.current_education,
    ci.test_scores, ci.message, ci.resume_url,
    ci.status, ci.reviewer_id, ci.reviewed_at, ci.review_note,
    ci.created_at, ci.updated_at,
    COALESCE(u.name, co.name, '') AS target_name
FROM counselling_inquiries ci
LEFT JOIN universities u
    ON ci.target_type = 'university' AND u.id = ci.target_id
LEFT JOIN colleges co
    ON ci.target_type = 'college' AND co.id = ci.target_id
WHERE (NULLIF($1, '') IS NULL OR ci.status = $1)
  AND (
        (NULLIF($2, '') IS NULL)
     OR ($2 = '__general__' AND ci.target_type IS NULL)
     OR ($2 <> '__general__' AND ci.target_type = $2
            AND (NULLIF($3, '') IS NULL OR ci.target_id::text = $3))
  )
ORDER BY ci.created_at DESC
LIMIT $4 OFFSET $5;

-- name: UpdateCounsellingStatus :one
-- Sets status + reviewer stamp + note in one shot. The service decides
-- whether to set reviewed_at (= NOW() when transitioning to reviewed,
-- stays NULL for archived if the row was never reviewed, etc.).
UPDATE counselling_inquiries
SET status = $2,
    reviewer_id = $3,
    reviewed_at = $4,
    review_note = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;
