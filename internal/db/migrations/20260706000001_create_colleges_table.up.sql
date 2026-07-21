CREATE TABLE IF NOT EXISTS colleges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    university_id UUID NOT NULL REFERENCES universities(id) ON DELETE RESTRICT,
    overview TEXT NOT NULL,
    country VARCHAR(100),
    state VARCHAR(100),
    city VARCHAR(100),
    full_location VARCHAR(255),
    logo VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_colleges_university_id ON colleges(university_id);
CREATE INDEX IF NOT EXISTS idx_colleges_name_trgm ON colleges USING gin (name gin_trgm_ops);