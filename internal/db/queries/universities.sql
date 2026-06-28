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
    contact_email, contact_phone, website
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19,
    $20, $21, $22, $23, $24, $25, $26, $27, $28
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