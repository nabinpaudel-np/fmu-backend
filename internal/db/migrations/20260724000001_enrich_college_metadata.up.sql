ALTER TABLE colleges
    ADD COLUMN excerpt VARCHAR(500),
    ADD COLUMN contact_email VARCHAR(255),
    ADD COLUMN contact_phone VARCHAR(50),
    ADD COLUMN website VARCHAR(500),
    ADD COLUMN zipcode VARCHAR(20),
    ADD COLUMN cover_image VARCHAR(500),
    ADD COLUMN gallery_images TEXT[],
    ADD COLUMN institution_type VARCHAR(50),
    ADD COLUMN campus_setting VARCHAR(50),
    ADD COLUMN founded_year SMALLINT,
    ADD COLUMN campus_size VARCHAR(100),
    ADD COLUMN is_popular BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN is_featured BOOLEAN NOT NULL DEFAULT FALSE;
