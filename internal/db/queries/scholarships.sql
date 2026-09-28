-- name: CreateScholarship :one
INSERT INTO scholarships (
    title, slug, description, award_amount, award_min, award_max, logo,
    number_of_awards, is_renewable,
    min_gpa, requires_financial_need, country, state, city,
    application_open_date, application_deadline, award_notification_date,
    essay_required, essay_prompt, recommendation_letters_required,
    transcript_requirement, portfolio_required, application_url,
    provider_type, provider_name, contact_email, contact_phone, internal_notes,
    university_id, college_id,
    seo_title, seo_description,
    is_popular, is_featured, status
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
    $21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
    $31, $32, $33, $34, $35
)
RETURNING *;

-- name: GetScholarshipByID :one
SELECT * FROM scholarships WHERE id = $1;

-- name: DeleteScholarship :execrows
DELETE FROM scholarships WHERE id = $1;

-- name: PublishScholarship :one
UPDATE scholarships
SET status = 'published',
    published_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SearchScholarships :many
-- Search over title: a case-insensitive substring match (so a short query like
-- "stem" finds "STEM Excellence Scholarship") OR typo-tolerant pg_trgm
-- similarity (threshold 0.2, below the PG default of 0.3, so near-misses still
-- match). Both forms are index-accelerated by the trigram GIN index on title.
-- Only published rows are searchable; COALESCE keeps nullable columns as
-- plain strings.
SELECT
    id,
    title,
    slug,
    COALESCE(provider_name, '') AS provider_name,
    COALESCE(provider_type, '') AS provider_type,
    COALESCE(logo, '') AS logo,
    award_amount
FROM scholarships
WHERE status = 'published'
  AND (title ILIKE '%' || sqlc.arg(query) || '%' OR similarity(title, sqlc.arg(query)) > 0.2)
ORDER BY similarity(title, sqlc.arg(query)) DESC, title ASC
LIMIT sqlc.arg(result_limit);

-- name: GetExistingEducationLevelIDs :many
SELECT id FROM education_levels WHERE id = ANY($1::uuid[]);

-- name: GetExistingDemographicIDs :many
SELECT id FROM demographics WHERE id = ANY($1::uuid[]);

-- name: InsertScholarshipEducationLevels :exec
INSERT INTO scholarship_education_levels (scholarship_id, education_level_id)
SELECT $1, unnest($2::uuid[])
ON CONFLICT (scholarship_id, education_level_id) DO NOTHING;

-- name: InsertScholarshipMajors :exec
INSERT INTO scholarship_majors (scholarship_id, major_id)
SELECT $1, unnest($2::uuid[])
ON CONFLICT (scholarship_id, major_id) DO NOTHING;

-- name: InsertScholarshipDemographics :exec
INSERT INTO scholarship_demographics (scholarship_id, demographic_id)
SELECT $1, unnest($2::uuid[])
ON CONFLICT (scholarship_id, demographic_id) DO NOTHING;

-- name: DeleteScholarshipEducationLevels :exec
DELETE FROM scholarship_education_levels WHERE scholarship_id = $1;

-- name: DeleteScholarshipMajors :exec
DELETE FROM scholarship_majors WHERE scholarship_id = $1;

-- name: DeleteScholarshipDemographics :exec
DELETE FROM scholarship_demographics WHERE scholarship_id = $1;

-- name: GetScholarshipEducationLevels :many
SELECT el.id, el.name
FROM education_levels el
JOIN scholarship_education_levels sel ON el.id = sel.education_level_id
WHERE sel.scholarship_id = $1
ORDER BY el.name;

-- name: GetScholarshipMajors :many
SELECT m.id, m.name
FROM majors m
JOIN scholarship_majors sm ON m.id = sm.major_id
WHERE sm.scholarship_id = $1
ORDER BY m.name;

-- name: GetScholarshipDemographics :many
SELECT d.id, d.name
FROM demographics d
JOIN scholarship_demographics sd ON d.id = sd.demographic_id
WHERE sd.scholarship_id = $1
ORDER BY d.name;

-- name: GetEducationLevels :many
SELECT id, name FROM education_levels ORDER BY name;

-- name: GetDemographics :many
SELECT id, name FROM demographics ORDER BY name;
