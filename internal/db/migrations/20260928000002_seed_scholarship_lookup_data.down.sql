DELETE FROM education_levels WHERE name IN (
    'High School Senior',
    'Undergraduate',
    'Graduate',
    'Ph.D.'
);

DELETE FROM demographics WHERE name IN (
    'First-Generation',
    'Women',
    'Minority / BIPOC',
    'Military / Veteran',
    'LGBTQ+',
    'Students with Disabilities',
    'International Students'
);
