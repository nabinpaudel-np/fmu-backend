-- College profile scalars (mirror university profile fields).
-- Nullable, no default — UI may fill any subset; the publish endpoint does
-- not require any of them to be set, and the create-time draft shortcut
-- (name + slug only) keeps working as before.
ALTER TABLE colleges
    ADD COLUMN full_address VARCHAR(500),
    ADD COLUMN maps_url VARCHAR(500),
    ADD COLUMN seo_title VARCHAR(70),
    ADD COLUMN seo_description VARCHAR(160);

-- Many-to-many: each college ↔ lookup row lives in its own junction table
-- with a composite PK. FKs use ON DELETE CASCADE so deleting a college (or
-- a lookup row) cleans up the association table automatically.
CREATE TABLE IF NOT EXISTS college_degree_levels (
    college_id UUID NOT NULL REFERENCES colleges(id) ON DELETE CASCADE,
    degree_level_id UUID NOT NULL REFERENCES degree_levels(id) ON DELETE CASCADE,
    PRIMARY KEY (college_id, degree_level_id)
);

CREATE TABLE IF NOT EXISTS college_majors (
    college_id UUID NOT NULL REFERENCES colleges(id) ON DELETE CASCADE,
    major_id UUID NOT NULL REFERENCES majors(id) ON DELETE CASCADE,
    PRIMARY KEY (college_id, major_id)
);

CREATE TABLE IF NOT EXISTS college_study_formats (
    college_id UUID NOT NULL REFERENCES colleges(id) ON DELETE CASCADE,
    study_format_id UUID NOT NULL REFERENCES study_formats(id) ON DELETE CASCADE,
    PRIMARY KEY (college_id, study_format_id)
);