-- migrate:up
ALTER TABLE users ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'student';

-- migrate:down
ALTER TABLE users DROP COLUMN role;
