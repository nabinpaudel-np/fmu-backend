DROP INDEX IF EXISTS idx_colleges_status;
ALTER TABLE colleges DROP CONSTRAINT IF EXISTS colleges_status_check;
ALTER TABLE colleges DROP COLUMN IF EXISTS published_at;
ALTER TABLE colleges DROP COLUMN IF EXISTS status;

DROP INDEX IF EXISTS idx_universities_status;
ALTER TABLE universities DROP CONSTRAINT IF EXISTS universities_status_check;
ALTER TABLE universities DROP COLUMN IF EXISTS published_at;
ALTER TABLE universities DROP COLUMN IF EXISTS status;