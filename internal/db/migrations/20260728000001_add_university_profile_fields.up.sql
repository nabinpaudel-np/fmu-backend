ALTER TABLE universities
    ADD COLUMN maps_url VARCHAR(500),
    ADD COLUMN full_address VARCHAR(500),
    ADD COLUMN employment_rate NUMERIC(5,2),
    ADD COLUMN research_output VARCHAR(50),
    ADD COLUMN housing_type VARCHAR(50),
    ADD COLUMN seo_title VARCHAR(70),
    ADD COLUMN seo_description VARCHAR(160);