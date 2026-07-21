CREATE TABLE university_favorites (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    university_id UUID NOT NULL REFERENCES universities(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, university_id)
);

CREATE INDEX idx_university_favorites_user_created
    ON university_favorites(user_id, created_at DESC);

CREATE TABLE college_favorites (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    college_id UUID NOT NULL REFERENCES colleges(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, college_id)
);

CREATE INDEX idx_college_favorites_user_created
    ON college_favorites(user_id, created_at DESC);