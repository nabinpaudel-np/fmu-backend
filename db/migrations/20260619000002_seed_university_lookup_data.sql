-- migrate:up

-- =====================================================
-- SEED: Degree Levels
-- =====================================================
INSERT INTO degree_levels (name) VALUES
    ('Certificate'),
    ('Associate'),
    ('Bachelor''s'),
    ('Master''s'),
    ('Doctorate')
ON CONFLICT (name) DO NOTHING;

-- =====================================================
-- SEED: Majors
-- =====================================================
INSERT INTO majors (name) VALUES
    ('Computer Science'),
    ('Business'),
    ('Engineering'),
    ('Medicine'),
    ('Biology'),
    ('Psychology'),
    ('Economics'),
    ('Art & Design'),
    ('Law'),
    ('Nursing')
ON CONFLICT (name) DO NOTHING;

-- =====================================================
-- SEED: Study Formats
-- =====================================================
INSERT INTO study_formats (name) VALUES
    ('100% In-Person'),
    ('100% Online'),
    ('Hybrid / Blended')
ON CONFLICT (name) DO NOTHING;

-- =====================================================
-- SEED: Special Affiliations
-- =====================================================
INSERT INTO special_affiliations (name) VALUES
    ('HBCU'),
    ('HSI'),
    ('Women''s College'),
    ('Men''s College')
ON CONFLICT (name) DO NOTHING;

-- =====================================================
-- SEED: Athletics
-- =====================================================
INSERT INTO athletics (name) VALUES
    ('NCAA Division I'),
    ('NCAA Division II'),
    ('NCAA Division III'),
    ('NAIA'),
    ('Intramural Sports')
ON CONFLICT (name) DO NOTHING;

-- =====================================================
-- SEED: Support Services
-- =====================================================
INSERT INTO support_services (name) VALUES
    ('Greek Life'),
    ('ROTC'),
    ('Marching Band'),
    ('Veteran Services'),
    ('Disability Services'),
    ('LGBTQ+ Support'),
    ('International Student Center')
ON CONFLICT (name) DO NOTHING;

-- migrate:down

DELETE FROM support_services WHERE name IN (
    'Greek Life', 'ROTC', 'Marching Band', 'Veteran Services',
    'Disability Services', 'LGBTQ+ Support', 'International Student Center'
);

DELETE FROM athletics WHERE name IN (
    'NCAA Division I', 'NCAA Division II', 'NCAA Division III', 'NAIA', 'Intramural Sports'
);

DELETE FROM special_affiliations WHERE name IN (
    'HBCU', 'HSI', 'Women''s College', 'Men''s College'
);

DELETE FROM study_formats WHERE name IN (
    '100% In-Person', '100% Online', 'Hybrid / Blended'
);

DELETE FROM majors WHERE name IN (
    'Computer Science', 'Business', 'Engineering', 'Medicine', 'Biology',
    'Psychology', 'Economics', 'Art & Design', 'Law', 'Nursing'
);

DELETE FROM degree_levels WHERE name IN (
    'Certificate', 'Associate', 'Bachelor''s', 'Master''s', 'Doctorate'
);
