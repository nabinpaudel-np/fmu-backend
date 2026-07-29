ALTER TABLE programs
    ADD COLUMN excerpt TEXT,
    ADD COLUMN career_options TEXT;

-- Backfill the existing seeded programs so the new fields aren't NULL
-- on environments that ran seed_programs before these columns existed.
UPDATE programs SET
    excerpt = CASE title
        WHEN 'BCA'            THEN 'Software development, databases, and web technologies — three years, hands-on from day one.'
        WHEN 'CSIT'           THEN 'Software engineering, networking, and information systems in a four-year integrated track.'
        WHEN 'CS'             THEN 'Algorithms, programming languages, and computational theory with strong math foundations.'
        WHEN 'IT'             THEN 'IT infrastructure, systems administration, and enterprise application delivery.'
        WHEN 'Software Eng'   THEN 'Software design, testing, and project management across the full SDLC.'
        WHEN 'Business'       THEN 'Management, marketing, finance, and entrepreneurship fundamentals.'
        WHEN 'MBA'            THEN 'Business leadership, strategy, and operations for working professionals and recent graduates.'
        WHEN 'Data Science'   THEN 'Statistics, machine learning, and data engineering with real datasets.'
        ELSE excerpt
    END,
    career_options = CASE title
        WHEN 'BCA'            THEN 'Software developer · Web developer · Database administrator · QA engineer'
        WHEN 'CSIT'           THEN 'Systems analyst · Network engineer · Software developer · IT consultant'
        WHEN 'CS'             THEN 'Software engineer · Data scientist · Research scientist · ML engineer'
        WHEN 'IT'             THEN 'Systems administrator · IT support lead · Cloud engineer · Enterprise architect'
        WHEN 'Software Eng'   THEN 'Software engineer · DevOps engineer · QA lead · Engineering manager'
        WHEN 'Business'       THEN 'Business analyst · Marketing associate · Operations coordinator · Entrepreneur'
        WHEN 'MBA'            THEN 'Product manager · Strategy consultant · Operations manager · Business analyst'
        WHEN 'Data Science'   THEN 'Data analyst · Data scientist · ML engineer · BI analyst'
        ELSE career_options
    END;
