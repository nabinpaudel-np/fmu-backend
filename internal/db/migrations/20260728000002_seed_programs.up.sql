ALTER TABLE programs
    ADD CONSTRAINT programs_title_key UNIQUE (title);

INSERT INTO programs (title, description, degree_id)
VALUES
    ('BCA', 'Bachelor of Computer Application — undergraduate program covering software development, databases, and web technologies.',
        (SELECT id FROM degree_levels WHERE name = 'Bachelor''s')),
    ('CSIT', 'Bachelor of Computer Science and Information Technology — undergraduate program in software engineering, networking, and information systems.',
        (SELECT id FROM degree_levels WHERE name = 'Bachelor''s')),
    ('CS', 'Bachelor of Computer Science — undergraduate program in algorithms, programming languages, and computational theory.',
        (SELECT id FROM degree_levels WHERE name = 'Bachelor''s')),
    ('IT', 'Bachelor of Information Technology — undergraduate program focused on IT infrastructure, systems administration, and enterprise applications.',
        (SELECT id FROM degree_levels WHERE name = 'Bachelor''s')),
    ('Software Eng', 'Bachelor of Software Engineering — undergraduate program in software design, testing, and project management.',
        (SELECT id FROM degree_levels WHERE name = 'Bachelor''s')),
    ('Business', 'Bachelor of Business — undergraduate program in management, marketing, finance, and entrepreneurship.',
        (SELECT id FROM degree_levels WHERE name = 'Bachelor''s')),
    ('MBA', 'Master of Business Administration — graduate program in business leadership, strategy, and operations.',
        (SELECT id FROM degree_levels WHERE name = 'Master''s')),
    ('Data Science', 'Bachelor of Data Science — undergraduate program in statistics, machine learning, and data engineering.',
        (SELECT id FROM degree_levels WHERE name = 'Bachelor''s'))
ON CONFLICT (title) DO NOTHING;