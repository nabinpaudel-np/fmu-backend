-- Natural-language search matches programme titles by similarity, so the
-- trigram index is what keeps "computr scienc" from scanning the table.
-- The extension is created by an earlier migration; this one only adds the
-- index.

CREATE INDEX IF NOT EXISTS idx_programs_title_trgm
    ON programs USING gin (title gin_trgm_ops);

-- The natural-language search reaches a programme through its degree level,
-- and a university through the degree levels it awards, so both sides of
-- that join are indexed.
CREATE INDEX IF NOT EXISTS idx_university_degree_levels_degree_level_id
    ON university_degree_levels (degree_level_id);

CREATE INDEX IF NOT EXISTS idx_college_degree_levels_degree_level_id
    ON college_degree_levels (degree_level_id);
