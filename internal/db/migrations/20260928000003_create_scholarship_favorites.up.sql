CREATE TABLE IF NOT EXISTS scholarship_favorites (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scholarship_id UUID NOT NULL REFERENCES scholarships(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, scholarship_id)
);

CREATE INDEX idx_scholarship_favorites_user_created
    ON scholarship_favorites(user_id, created_at DESC);
