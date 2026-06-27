-- =====================================================
-- UNIVERSITIES (core table)
-- =====================================================
CREATE TABLE IF NOT EXISTS universities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Basic Info
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    overview TEXT,
    excerpt VARCHAR(500),

    -- Location
    country VARCHAR(100),
    state VARCHAR(100),
    city VARCHAR(100),
    full_location VARCHAR(255),
    cover_image VARCHAR(500),
    logo VARCHAR(500),

    -- Academics (single-select)
    institution_type VARCHAR(50),
    campus_setting VARCHAR(50),

    -- Tuition
    in_state_tuition DECIMAL(12,2),
    out_of_state_tuition DECIMAL(12,2),
    international_tuition DECIMAL(12,2),

    -- Financial Aid (booleans)
    need_based_aid BOOLEAN NOT NULL DEFAULT FALSE,
    merit_scholarships BOOLEAN NOT NULL DEFAULT FALSE,
    work_study BOOLEAN NOT NULL DEFAULT FALSE,
    no_application_fee BOOLEAN NOT NULL DEFAULT FALSE,

    -- Admissions
    acceptance_rate DECIMAL(5,2),
    testing_policy VARCHAR(50),
    sat_range VARCHAR(20),
    act_range VARCHAR(20),
    on_campus_housing BOOLEAN NOT NULL DEFAULT FALSE,
    freshmen_required_on_campus BOOLEAN NOT NULL DEFAULT FALSE,

    -- Contact
    contact_email VARCHAR(255),
    contact_phone VARCHAR(50),
    website VARCHAR(500),

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_universities_slug ON universities(slug);
CREATE INDEX idx_universities_name ON universities(name);

-- =====================================================
-- LOOKUP TABLES
-- =====================================================

CREATE TABLE IF NOT EXISTS degree_levels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS majors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS study_formats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS special_affiliations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS athletics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS support_services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE
);

-- =====================================================
-- JUNCTION TABLES (many-to-many)
-- =====================================================

CREATE TABLE IF NOT EXISTS university_degree_levels (
    university_id UUID NOT NULL REFERENCES universities(id) ON DELETE CASCADE,
    degree_level_id UUID NOT NULL REFERENCES degree_levels(id) ON DELETE CASCADE,
    PRIMARY KEY (university_id, degree_level_id)
);

CREATE TABLE IF NOT EXISTS university_majors (
    university_id UUID NOT NULL REFERENCES universities(id) ON DELETE CASCADE,
    major_id UUID NOT NULL REFERENCES majors(id) ON DELETE CASCADE,
    PRIMARY KEY (university_id, major_id)
);

CREATE TABLE IF NOT EXISTS university_study_formats (
    university_id UUID NOT NULL REFERENCES universities(id) ON DELETE CASCADE,
    study_format_id UUID NOT NULL REFERENCES study_formats(id) ON DELETE CASCADE,
    PRIMARY KEY (university_id, study_format_id)
);

CREATE TABLE IF NOT EXISTS university_special_affiliations (
    university_id UUID NOT NULL REFERENCES universities(id) ON DELETE CASCADE,
    special_affiliation_id UUID NOT NULL REFERENCES special_affiliations(id) ON DELETE CASCADE,
    PRIMARY KEY (university_id, special_affiliation_id)
);

CREATE TABLE IF NOT EXISTS university_athletics (
    university_id UUID NOT NULL REFERENCES universities(id) ON DELETE CASCADE,
    athletic_id UUID NOT NULL REFERENCES athletics(id) ON DELETE CASCADE,
    PRIMARY KEY (university_id, athletic_id)
);

CREATE TABLE IF NOT EXISTS university_support_services (
    university_id UUID NOT NULL REFERENCES universities(id) ON DELETE CASCADE,
    support_service_id UUID NOT NULL REFERENCES support_services(id) ON DELETE CASCADE,
    PRIMARY KEY (university_id, support_service_id)
);