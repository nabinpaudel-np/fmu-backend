-- name: CreateCollege :one
INSERT INTO colleges (
    name, slug, university_id, overview, excerpt,
    country, state, city, full_location,
    cover_image, logo, institution_type, campus_setting,
    contact_email, contact_phone, website, zipcode,
    founded_year, campus_size, gallery_images,
    is_popular, is_featured,
    full_address, maps_url, seo_title, seo_description,
    status
)
VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11, $12, $13,
    $14, $15, $16, $17,
    $18, $19, $20,
    $21, $22,
    $23, $24, $25, $26,
    $27
)
RETURNING
    id, name, slug, university_id, overview, excerpt,
    country, state, city, full_location,
    cover_image, logo, institution_type, campus_setting,
    contact_email, contact_phone, website, zipcode,
    founded_year, campus_size, gallery_images,
    is_popular, is_featured,
    full_address, maps_url, seo_title, seo_description,
    status, published_at, created_at, updated_at;

-- name: GetCollegeByID :one
SELECT
    id, name, slug, university_id, overview, excerpt,
    country, state, city, full_location,
    cover_image, logo, institution_type, campus_setting,
    contact_email, contact_phone, website, zipcode,
    founded_year, campus_size, gallery_images,
    is_popular, is_featured,
    full_address, maps_url, seo_title, seo_description,
    status, published_at, created_at, updated_at
FROM colleges
WHERE id = $1;

-- name: ListCollegesByUniversity :many
SELECT
    id, name, slug, university_id, overview, excerpt,
    country, state, city, full_location,
    cover_image, logo, institution_type, campus_setting,
    contact_email, contact_phone, website, zipcode,
    founded_year, campus_size, gallery_images,
    is_popular, is_featured,
    full_address, maps_url, seo_title, seo_description,
    status, published_at, created_at, updated_at
FROM colleges
WHERE university_id = $1
ORDER BY name
LIMIT $2 OFFSET $3;

-- name: CountCollegesByUniversity :one
SELECT COUNT(*) FROM colleges WHERE university_id = $1;

-- name: GetExistingCollegeDegreeLevelIDs :many
SELECT id FROM degree_levels WHERE id = ANY($1::uuid[]);

-- name: GetExistingCollegeMajorIDs :many
SELECT id FROM majors WHERE id = ANY($1::uuid[]);

-- name: GetExistingCollegeStudyFormatIDs :many
SELECT id FROM study_formats WHERE id = ANY($1::uuid[]);

-- name: InsertCollegeDegreeLevels :exec
INSERT INTO college_degree_levels (college_id, degree_level_id)
SELECT $1, unnest($2::uuid[])
ON CONFLICT (college_id, degree_level_id) DO NOTHING;

-- name: InsertCollegeMajors :exec
INSERT INTO college_majors (college_id, major_id)
SELECT $1, unnest($2::uuid[])
ON CONFLICT (college_id, major_id) DO NOTHING;

-- name: InsertCollegeStudyFormats :exec
INSERT INTO college_study_formats (college_id, study_format_id)
SELECT $1, unnest($2::uuid[])
ON CONFLICT (college_id, study_format_id) DO NOTHING;

-- name: DeleteCollegeDegreeLevels :exec
DELETE FROM college_degree_levels WHERE college_id = $1;

-- name: DeleteCollegeMajors :exec
DELETE FROM college_majors WHERE college_id = $1;

-- name: DeleteCollegeStudyFormats :exec
DELETE FROM college_study_formats WHERE college_id = $1;

-- name: GetCollegeDegreeLevels :many
SELECT dl.id, dl.name
FROM degree_levels dl
JOIN college_degree_levels cdl ON dl.id = cdl.degree_level_id
WHERE cdl.college_id = $1
ORDER BY dl.name;

-- name: GetCollegeMajors :many
SELECT m.id, m.name
FROM majors m
JOIN college_majors cm ON m.id = cm.major_id
WHERE cm.college_id = $1
ORDER BY m.name;

-- name: GetCollegeStudyFormats :many
SELECT sf.id, sf.name
FROM study_formats sf
JOIN college_study_formats csf ON sf.id = csf.study_format_id
WHERE csf.college_id = $1
ORDER BY sf.name;

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
WHERE c.status = 'published'
  AND (similarity(c.name, $1) > 0.2
   OR similarity(c.full_location, $1) > 0.2
   OR similarity(c.city, $1) > 0.2
   OR similarity(c.state, $1) > 0.2
   OR similarity(c.country, $1) > 0.2
   OR similarity(u.name, $1) > 0.2
   OR similarity(u.slug, $1) > 0.2)
ORDER BY GREATEST(
    similarity(c.name, $1),
    similarity(c.full_location, $1),
    similarity(c.city, $1),
    similarity(u.name, $1),
    similarity(u.slug, $1)
) DESC, c.name ASC
LIMIT $2;

-- name: ListRepresentedCollegeIDs :many
SELECT c.id
FROM colleges c
JOIN users u ON u.representative_college_id = c.id
WHERE c.id = ANY($1::uuid[]);

-- name: PublishCollege :one
UPDATE colleges
SET status = 'published',
    published_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING id, name, slug, university_id, overview, excerpt,
    country, state, city, full_location,
    cover_image, logo, institution_type, campus_setting,
    contact_email, contact_phone, website, zipcode,
    founded_year, campus_size, gallery_images,
    is_popular, is_featured,
    full_address, maps_url, seo_title, seo_description,
    status, published_at, created_at, updated_at;