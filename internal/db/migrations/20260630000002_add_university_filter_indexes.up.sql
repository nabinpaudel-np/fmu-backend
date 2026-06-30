-- Filter-friendly indexes for `GET /api/v1/universities`.
-- The join-table reverse indexes accelerate EXISTS subqueries built by the
-- repository for the multi-value lookup filters (majors, degree_levels, etc.).

CREATE INDEX IF NOT EXISTS idx_universities_country
    ON universities (country);

CREATE INDEX IF NOT EXISTS idx_universities_state
    ON universities (state);

CREATE INDEX IF NOT EXISTS idx_universities_city
    ON universities (city);

CREATE INDEX IF NOT EXISTS idx_universities_institution_type
    ON universities (institution_type);

CREATE INDEX IF NOT EXISTS idx_universities_campus_setting
    ON universities (campus_setting);

CREATE INDEX IF NOT EXISTS idx_universities_testing_policy
    ON universities (testing_policy);

CREATE INDEX IF NOT EXISTS idx_universities_tuition_min
    ON universities (tuition_min);

CREATE INDEX IF NOT EXISTS idx_universities_tuition_max
    ON universities (tuition_max);

CREATE INDEX IF NOT EXISTS idx_universities_acceptance_rate
    ON universities (acceptance_rate);

CREATE INDEX IF NOT EXISTS idx_universities_is_popular
    ON universities (is_popular);

CREATE INDEX IF NOT EXISTS idx_universities_is_featured
    ON universities (is_featured);

-- Reverse indexes on each join table: each multi-value filter ("has ANY of
-- these majors") resolves to a `EXISTS (SELECT 1 ... WHERE lookup_id = $1)`
-- subquery, and the planner hits these covering indexes.

CREATE INDEX IF NOT EXISTS idx_university_majors_major_id
    ON university_majors (major_id);

CREATE INDEX IF NOT EXISTS idx_university_degree_levels_degree_level_id
    ON university_degree_levels (degree_level_id);

CREATE INDEX IF NOT EXISTS idx_university_study_formats_study_format_id
    ON university_study_formats (study_format_id);

CREATE INDEX IF NOT EXISTS idx_university_special_affiliations_special_affiliation_id
    ON university_special_affiliations (special_affiliation_id);

CREATE INDEX IF NOT EXISTS idx_university_athletics_athletic_id
    ON university_athletics (athletic_id);

CREATE INDEX IF NOT EXISTS idx_university_support_services_support_service_id
    ON university_support_services (support_service_id);
