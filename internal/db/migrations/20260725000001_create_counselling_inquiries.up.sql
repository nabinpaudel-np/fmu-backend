CREATE TABLE IF NOT EXISTS counselling_inquiries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type VARCHAR(20),
    target_id UUID,
    full_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    country VARCHAR(100),
    preferred_university VARCHAR(255),
    program_of_interest VARCHAR(255),
    start_term VARCHAR(100),
    current_education VARCHAR(100),
    test_scores TEXT,
    message TEXT,
    resume_url VARCHAR(500),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    reviewer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    review_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT counselling_inquiries_target_type_check
        CHECK (target_type IS NULL OR target_type IN ('university', 'college')),
    CONSTRAINT counselling_inquiries_target_pair_check
        CHECK ((target_type IS NULL AND target_id IS NULL)
            OR (target_type IS NOT NULL AND target_id IS NOT NULL)),
    CONSTRAINT counselling_inquiries_status_check
        CHECK (status IN ('pending', 'reviewed', 'archived'))
);

-- target_type+target_id is the primary access path for representative dashboards
-- (filter "everything for my university") and admin filters ("only colleges").
CREATE INDEX idx_counselling_inquiries_target
    ON counselling_inquiries(target_type, target_id, created_at DESC)
    WHERE target_type IS NOT NULL;

-- General-only feed + status filter for the admin overview.
CREATE INDEX idx_counselling_inquiries_status_created
    ON counselling_inquiries(status, created_at DESC);

CREATE INDEX idx_counselling_inquiries_created_at
    ON counselling_inquiries(created_at DESC);
