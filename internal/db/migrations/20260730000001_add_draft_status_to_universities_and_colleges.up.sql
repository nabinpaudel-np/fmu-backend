ALTER TABLE universities
    ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'published',
    ADD COLUMN published_at TIMESTAMPTZ NULL;

ALTER TABLE universities
    ADD CONSTRAINT universities_status_check
    CHECK (status IN ('draft', 'published', 'archived'));

CREATE INDEX idx_universities_status ON universities(status);

ALTER TABLE colleges
    ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'published',
    ADD COLUMN published_at TIMESTAMPTZ NULL;

ALTER TABLE colleges
    ADD CONSTRAINT colleges_status_check
    CHECK (status IN ('draft', 'published', 'archived'));

CREATE INDEX idx_colleges_status ON colleges(status);