CREATE TABLE IF NOT EXISTS college_claims (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    college_id UUID NOT NULL REFERENCES colleges(id) ON DELETE CASCADE,
    full_name VARCHAR(255) NOT NULL,
    work_email VARCHAR(255) NOT NULL,
    document_url VARCHAR(500) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    reviewer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    review_note TEXT,
    created_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT college_claims_status_check
        CHECK (status IN ('pending', 'approved', 'rejected'))
);

CREATE INDEX idx_college_claims_college_id ON college_claims(college_id);
CREATE INDEX idx_college_claims_status ON college_claims(status, created_at DESC);

ALTER TABLE users
    ADD COLUMN representative_college_id UUID UNIQUE REFERENCES colleges(id) ON DELETE SET NULL;

CREATE INDEX idx_users_representative_college_id
    ON users(representative_college_id)
    WHERE representative_college_id IS NOT NULL;