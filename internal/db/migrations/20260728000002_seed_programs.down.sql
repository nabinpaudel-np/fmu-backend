DELETE FROM programs WHERE title IN (
    'BCA',
    'CSIT',
    'CS',
    'IT',
    'Software Eng',
    'Business',
    'MBA',
    'Data Science'
);

ALTER TABLE programs
    DROP CONSTRAINT IF EXISTS programs_title_key;