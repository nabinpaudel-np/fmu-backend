ALTER TABLE users
    ADD COLUMN representative_university_id UUID UNIQUE REFERENCES universities(id) ON DELETE SET NULL;

CREATE INDEX idx_users_representative_university_id
    ON users(representative_university_id)
    WHERE representative_university_id IS NOT NULL;