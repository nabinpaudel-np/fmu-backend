-- Trigram indexes for typo-tolerant search across name + location fields.
-- The pg_trgm extension is bundled with the official postgres docker image
-- we run in dev/prod (postgres:17). If this fails on a vanilla install,
-- install the postgres-contrib package, not change this migration.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- gin_trgm_ops supports both ILIKE '%x%' and the similarity() function
-- used by the /api/v1/universities/search endpoint.
CREATE INDEX IF NOT EXISTS idx_universities_name_trgm
    ON universities USING gin (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_universities_full_location_trgm
    ON universities USING gin (full_location gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_universities_city_trgm
    ON universities USING gin (city gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_universities_state_trgm
    ON universities USING gin (state gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_universities_country_trgm
    ON universities USING gin (country gin_trgm_ops);
