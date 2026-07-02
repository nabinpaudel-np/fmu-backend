-- name: CreateUniversity :one
INSERT INTO universities (
    name, slug, overview, excerpt,
    country, state, city, full_location,
    cover_image, logo,
    institution_type, campus_setting,
    in_state_tuition, out_of_state_tuition, international_tuition,
    need_based_aid, merit_scholarships, work_study, no_application_fee,
    acceptance_rate, testing_policy, sat_range, act_range,
    on_campus_housing, freshmen_required_on_campus,
    contact_email, contact_phone, website,
    zipcode, tuition_min, tuition_max, avg_high_school_gpa,
    founded_year, campus_size, gallery_images,
    is_popular, is_featured
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19,
    $20, $21, $22, $23, $24, $25, $26, $27, $28,
    $29, $30, $31, $32, $33, $34, $35, $36, $37
)
RETURNING *;

-- name: GetExistingDegreeLevelIDs :many
SELECT id FROM degree_levels WHERE id = ANY($1::uuid[]);

-- name: GetExistingMajorIDs :many
SELECT id FROM majors WHERE id = ANY($1::uuid[]);

-- name: GetExistingStudyFormatIDs :many
SELECT id FROM study_formats WHERE id = ANY($1::uuid[]);

-- name: GetExistingSpecialAffiliationIDs :many
SELECT id FROM special_affiliations WHERE id = ANY($1::uuid[]);

-- name: GetExistingAthleticIDs :many
SELECT id FROM athletics WHERE id = ANY($1::uuid[]);

-- name: GetExistingSupportServiceIDs :many
SELECT id FROM support_services WHERE id = ANY($1::uuid[]);

-- name: InsertUniversityDegreeLevels :exec
INSERT INTO university_degree_levels (university_id, degree_level_id)
SELECT $1, unnest($2::uuid[])
ON CONFLICT (university_id, degree_level_id) DO NOTHING;

-- name: InsertUniversityMajors :exec
INSERT INTO university_majors (university_id, major_id)
SELECT $1, unnest($2::uuid[])
ON CONFLICT (university_id, major_id) DO NOTHING;

-- name: InsertUniversityStudyFormats :exec
INSERT INTO university_study_formats (university_id, study_format_id)
SELECT $1, unnest($2::uuid[])
ON CONFLICT (university_id, study_format_id) DO NOTHING;

-- name: InsertUniversitySpecialAffiliations :exec
INSERT INTO university_special_affiliations (university_id, special_affiliation_id)
SELECT $1, unnest($2::uuid[])
ON CONFLICT (university_id, special_affiliation_id) DO NOTHING;

-- name: InsertUniversityAthletics :exec
INSERT INTO university_athletics (university_id, athletic_id)
SELECT $1, unnest($2::uuid[])
ON CONFLICT (university_id, athletic_id) DO NOTHING;

-- name: InsertUniversitySupportServices :exec
INSERT INTO university_support_services (university_id, support_service_id)
SELECT $1, unnest($2::uuid[])
ON CONFLICT (university_id, support_service_id) DO NOTHING;

-- name: GetMajors :many
SELECT id, name FROM majors ORDER BY name;

-- name: ListUniversities :many
SELECT * FROM universities ORDER BY name LIMIT $1 OFFSET $2;

-- name: CountUniversities :one
SELECT COUNT(*) FROM universities;

-- name: GetUniversityByID :one
SELECT * FROM universities WHERE id = $1;

-- name: GetUniversityDegreeLevels :many
SELECT dl.id, dl.name
FROM degree_levels dl
JOIN university_degree_levels udl ON dl.id = udl.degree_level_id
WHERE udl.university_id = $1
ORDER BY dl.name;

-- name: GetUniversityMajors :many
SELECT m.id, m.name
FROM majors m
JOIN university_majors um ON m.id = um.major_id
WHERE um.university_id = $1
ORDER BY m.name;

-- name: GetUniversityStudyFormats :many
SELECT sf.id, sf.name
FROM study_formats sf
JOIN university_study_formats usf ON sf.id = usf.study_format_id
WHERE usf.university_id = $1
ORDER BY sf.name;

-- name: GetUniversitySpecialAffiliations :many
SELECT sa.id, sa.name
FROM special_affiliations sa
JOIN university_special_affiliations usa ON sa.id = usa.special_affiliation_id
WHERE usa.university_id = $1
ORDER BY sa.name;

-- name: GetUniversityAthletics :many
SELECT a.id, a.name
FROM athletics a
JOIN university_athletics ua ON a.id = ua.athletic_id
WHERE ua.university_id = $1
ORDER BY a.name;

-- name: GetUniversitySupportServices :many
SELECT ss.id, ss.name
FROM support_services ss
JOIN university_support_services uss ON ss.id = uss.support_service_id
WHERE uss.university_id = $1
ORDER BY ss.name;

-- name: GetDegreeLevels :many
SELECT id, name FROM degree_levels ORDER BY name;

-- name: GetStudyFormats :many
SELECT id, name FROM study_formats ORDER BY name;

-- name: GetSpecialAffiliations :many
SELECT id, name FROM special_affiliations ORDER BY name;

-- name: GetAthletics :many
SELECT id, name FROM athletics ORDER BY name;

-- name: GetSupportServices :many
SELECT id, name FROM support_services ORDER BY name;

-- name: SearchUniversities :many
-- Typo-tolerant search across name + location fields via pg_trgm similarity.
-- Threshold 0.2 is below the PG default of 0.3, so "cambrige" still matches
-- "Cambridge". Results are ranked by max similarity across the matched fields.
-- COALESCE keeps nullable location/URL columns as plain strings in the row type
-- (the API response does not carry nulls).
SELECT
    id,
    name,
    slug,
    COALESCE(country, '') AS country,
    COALESCE(state, '') AS state,
    COALESCE(city, '') AS city,
    COALESCE(full_location, '') AS full_location,
    COALESCE(logo, '') AS logo
FROM universities
WHERE similarity(name, $1) > 0.2
   OR similarity(full_location, $1) > 0.2
   OR similarity(city, $1) > 0.2
   OR similarity(state, $1) > 0.2
   OR similarity(country, $1) > 0.2
ORDER BY GREATEST(
    similarity(name, $1),
    similarity(full_location, $1),
    similarity(city, $1)
) DESC, name ASC
LIMIT $2;