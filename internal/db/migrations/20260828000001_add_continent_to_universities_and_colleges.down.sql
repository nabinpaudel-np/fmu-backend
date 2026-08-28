DROP INDEX IF EXISTS idx_colleges_continent_trgm;
DROP INDEX IF EXISTS idx_universities_continent_trgm;

ALTER TABLE colleges    DROP COLUMN IF EXISTS continent;
ALTER TABLE universities DROP COLUMN IF EXISTS continent;
