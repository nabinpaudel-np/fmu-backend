-- =====================================================
-- SEED: Education Levels
-- =====================================================
INSERT INTO education_levels (name) VALUES
    ('High School Senior'),
    ('Undergraduate'),
    ('Graduate'),
    ('Ph.D.')
ON CONFLICT (name) DO NOTHING;

-- =====================================================
-- SEED: Demographics
-- =====================================================
INSERT INTO demographics (name) VALUES
    ('First-Generation'),
    ('Women'),
    ('Minority / BIPOC'),
    ('Military / Veteran'),
    ('LGBTQ+'),
    ('Students with Disabilities'),
    ('International Students')
ON CONFLICT (name) DO NOTHING;
