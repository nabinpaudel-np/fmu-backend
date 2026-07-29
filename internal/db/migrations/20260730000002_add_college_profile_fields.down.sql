DROP TABLE IF EXISTS college_study_formats;
DROP TABLE IF EXISTS college_majors;
DROP TABLE IF EXISTS college_degree_levels;

ALTER TABLE colleges
    DROP COLUMN seo_description,
    DROP COLUMN seo_title,
    DROP COLUMN maps_url,
    DROP COLUMN full_address;