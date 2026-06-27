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