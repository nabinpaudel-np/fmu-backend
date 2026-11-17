DROP INDEX IF EXISTS idx_universities_search_vector;
DROP INDEX IF EXISTS idx_colleges_search_vector;
DROP INDEX IF EXISTS idx_programs_search_vector;

ALTER TABLE universities DROP COLUMN IF EXISTS search_vector;
ALTER TABLE colleges DROP COLUMN IF EXISTS search_vector;
ALTER TABLE programs DROP COLUMN IF EXISTS search_vector;
