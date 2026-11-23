-- Store a missing title or slug as '' instead of NULL, and forbid NULL. The
-- queries already treated NULL and '' the same, so this removes the conversions
-- between them (COALESCE when reading, NULLIF when saving).

-- +goose Up
UPDATE posts SET title = '' WHERE title IS NULL;
UPDATE posts SET slug = '' WHERE slug IS NULL;

ALTER TABLE posts
    ALTER COLUMN title SET DEFAULT '',
    ALTER COLUMN title SET NOT NULL,
    ALTER COLUMN slug SET DEFAULT '',
    ALTER COLUMN slug SET NOT NULL;
