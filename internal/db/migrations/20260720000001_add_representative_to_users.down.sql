DROP INDEX IF EXISTS idx_users_representative_university_id;

ALTER TABLE users
    DROP COLUMN IF EXISTS representative_university_id;