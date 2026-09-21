CREATE TABLE student_profile (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    budget BIGINT,
    intended_country VARCHAR(100),
    current_education VARCHAR(20),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT student_profile_current_education_check
        CHECK (current_education IS NULL OR current_education IN ('8-10', '10-12', 'bachelors', 'masters', 'phd'))
);

CREATE TABLE student_profile_programs (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    program_id UUID NOT NULL REFERENCES programs(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, program_id)
);

CREATE INDEX idx_student_profile_programs_user_id ON student_profile_programs(user_id);