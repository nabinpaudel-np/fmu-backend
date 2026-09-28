-- =====================================================
-- SCHOLARSHIPS (core table)
-- =====================================================
CREATE TABLE IF NOT EXISTS scholarships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Core details
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    award_amount VARCHAR(255) NOT NULL,
    award_min DECIMAL(12,2),
    award_max DECIMAL(12,2),
    logo VARCHAR(500),
    number_of_awards INTEGER,
    is_renewable BOOLEAN NOT NULL DEFAULT FALSE,

    -- Eligibility & targeting
    min_gpa DECIMAL(4,2),
    requires_financial_need BOOLEAN NOT NULL DEFAULT FALSE,
    country VARCHAR(100),
    state VARCHAR(100),
    city VARCHAR(100),

    -- Timeline & deadlines
    application_open_date TIMESTAMP WITH TIME ZONE,
    application_deadline TIMESTAMP WITH TIME ZONE,
    award_notification_date TIMESTAMP WITH TIME ZONE,

    -- Application requirements
    essay_required BOOLEAN NOT NULL DEFAULT FALSE,
    essay_prompt TEXT,
    recommendation_letters_required INTEGER NOT NULL DEFAULT 0,
    transcript_requirement VARCHAR(20) NOT NULL DEFAULT 'none',
    portfolio_required BOOLEAN NOT NULL DEFAULT FALSE,
    application_url VARCHAR(500),

    -- Provider information
    provider_type VARCHAR(50),
    provider_name VARCHAR(255),
    contact_email VARCHAR(255),
    contact_phone VARCHAR(50),
    -- internal_notes is admin-only; the API never exposes it to public callers.
    internal_notes TEXT,
    -- Optional links to an in-directory provider. Both nullable so external
    -- providers (Corporation / NGO / Government) can be stored via the
    -- free-text provider_* fields instead.
    university_id UUID REFERENCES universities(id) ON DELETE SET NULL,
    college_id UUID REFERENCES colleges(id) ON DELETE SET NULL,

    -- SEO
    seo_title VARCHAR(70),
    seo_description VARCHAR(160),

    -- Flags + lifecycle
    is_popular BOOLEAN NOT NULL DEFAULT FALSE,
    is_featured BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(20) NOT NULL DEFAULT 'published',
    published_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT scholarships_status_check
        CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT scholarships_transcript_requirement_check
        CHECK (transcript_requirement IN ('none', 'official', 'unofficial'))
);

CREATE INDEX idx_scholarships_status ON scholarships(status);
CREATE INDEX idx_scholarships_application_deadline ON scholarships(application_deadline);
CREATE INDEX idx_scholarships_provider_type ON scholarships(provider_type);
CREATE INDEX idx_scholarships_university_id ON scholarships(university_id);
CREATE INDEX idx_scholarships_college_id ON scholarships(college_id);
CREATE INDEX idx_scholarships_title_trgm ON scholarships USING gin (title gin_trgm_ops);

-- =====================================================
-- LOOKUP TABLES
-- =====================================================

CREATE TABLE IF NOT EXISTS education_levels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS demographics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE
);

-- =====================================================
-- JUNCTION TABLES (many-to-many)
-- =====================================================

CREATE TABLE IF NOT EXISTS scholarship_education_levels (
    scholarship_id UUID NOT NULL REFERENCES scholarships(id) ON DELETE CASCADE,
    education_level_id UUID NOT NULL REFERENCES education_levels(id) ON DELETE CASCADE,
    PRIMARY KEY (scholarship_id, education_level_id)
);

CREATE TABLE IF NOT EXISTS scholarship_majors (
    scholarship_id UUID NOT NULL REFERENCES scholarships(id) ON DELETE CASCADE,
    major_id UUID NOT NULL REFERENCES majors(id) ON DELETE CASCADE,
    PRIMARY KEY (scholarship_id, major_id)
);

CREATE TABLE IF NOT EXISTS scholarship_demographics (
    scholarship_id UUID NOT NULL REFERENCES scholarships(id) ON DELETE CASCADE,
    demographic_id UUID NOT NULL REFERENCES demographics(id) ON DELETE CASCADE,
    PRIMARY KEY (scholarship_id, demographic_id)
);

-- Reverse indexes on the lookup-id column for fast EXISTS filtering.
CREATE INDEX idx_scholarship_education_levels_education_level_id ON scholarship_education_levels(education_level_id);
CREATE INDEX idx_scholarship_majors_major_id ON scholarship_majors(major_id);
CREATE INDEX idx_scholarship_demographics_demographic_id ON scholarship_demographics(demographic_id);
