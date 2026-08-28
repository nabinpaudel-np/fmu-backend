ALTER TABLE universities ADD COLUMN continent VARCHAR(100);
ALTER TABLE colleges    ADD COLUMN continent VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_universities_continent_trgm ON universities USING gin (continent gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_colleges_continent_trgm    ON colleges    USING gin (continent gin_trgm_ops);
