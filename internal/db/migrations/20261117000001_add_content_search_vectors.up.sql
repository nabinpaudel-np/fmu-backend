-- Natural-language search matches the descriptive prose of a catalogue row,
-- not only its structured columns. These generated tsvector columns hold the
-- searchable text so the match can use an index instead of scanning.
--
-- The vector is generated rather than maintained by hand, so it can never
-- drift from the columns it is built from.

ALTER TABLE universities
    ADD COLUMN IF NOT EXISTS search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(overview, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(excerpt, '')), 'B')
    ) STORED;

ALTER TABLE colleges
    ADD COLUMN IF NOT EXISTS search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(overview, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(excerpt, '')), 'C')
    ) STORED;

ALTER TABLE programs
    ADD COLUMN IF NOT EXISTS search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(excerpt, '')), 'C') ||
        setweight(to_tsvector('english', coalesce(career_options, '')), 'C')
    ) STORED;

-- The GIN index is what keeps a content search from reading every row.
CREATE INDEX IF NOT EXISTS idx_universities_search_vector
    ON universities USING gin (search_vector);

CREATE INDEX IF NOT EXISTS idx_colleges_search_vector
    ON colleges USING gin (search_vector);

CREATE INDEX IF NOT EXISTS idx_programs_search_vector
    ON programs USING gin (search_vector);
