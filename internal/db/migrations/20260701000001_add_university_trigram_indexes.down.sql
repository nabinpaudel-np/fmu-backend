DROP INDEX IF EXISTS idx_universities_name_trgm;
DROP INDEX IF EXISTS idx_universities_full_location_trgm;
DROP INDEX IF EXISTS idx_universities_city_trgm;
DROP INDEX IF EXISTS idx_universities_state_trgm;
DROP INDEX IF EXISTS idx_universities_country_trgm;

DROP EXTENSION IF EXISTS pg_trgm;
