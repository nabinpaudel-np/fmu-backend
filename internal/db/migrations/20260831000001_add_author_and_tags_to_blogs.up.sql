ALTER TABLE public.blogs
    ADD COLUMN author_name        VARCHAR(255),
    ADD COLUMN author_description TEXT,
    ADD COLUMN author_title       VARCHAR(255),
    ADD COLUMN tags               TEXT[] NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_blogs_tags ON public.blogs USING GIN (tags);
