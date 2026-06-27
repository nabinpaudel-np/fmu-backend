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