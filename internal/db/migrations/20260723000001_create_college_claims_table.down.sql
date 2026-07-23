DROP INDEX IF EXISTS idx_users_representative_college_id;
ALTER TABLE users DROP COLUMN IF EXISTS representative_college_id;

DROP INDEX IF EXISTS idx_college_claims_status;
DROP INDEX IF EXISTS idx_college_claims_college_id;
DROP TABLE IF EXISTS college_claims;