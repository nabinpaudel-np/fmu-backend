DROP INDEX IF EXISTS idx_blogs_tags;
ALTER TABLE public.blogs
    DROP COLUMN IF EXISTS tags,
    DROP COLUMN IF EXISTS author_title,
    DROP COLUMN IF EXISTS author_description,
    DROP COLUMN IF EXISTS author_name;
