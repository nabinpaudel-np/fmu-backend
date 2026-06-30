ALTER TABLE universities
    ADD COLUMN zipcode VARCHAR(20),
    ADD COLUMN tuition_min INTEGER,
    ADD COLUMN tuition_max INTEGER,
    ADD COLUMN avg_high_school_gpa NUMERIC(4,2),
    ADD COLUMN founded_year SMALLINT,
    ADD COLUMN campus_size VARCHAR(100),
    ADD COLUMN gallery_images TEXT[],
    ADD COLUMN is_popular BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN is_featured BOOLEAN NOT NULL DEFAULT FALSE;