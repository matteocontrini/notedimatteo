-- Remove the default on time columns and truncate stored times to milliseconds.
-- The Go application sets every time, in milliseconds to match the SQLite format
-- (sqlite-plan.md). Before this migration, Go wrote microseconds.

-- +goose Up
ALTER TABLE posts
    ALTER COLUMN "createdAt" DROP DEFAULT,
    ALTER COLUMN "updatedAt" DROP DEFAULT,
    ALTER COLUMN "publishedAt" DROP DEFAULT;
ALTER TABLE sessions ALTER COLUMN "createdAt" DROP DEFAULT;
ALTER TABLE post_html_cache ALTER COLUMN "renderedAt" DROP DEFAULT;

UPDATE posts
SET "createdAt"   = date_trunc('milliseconds', "createdAt"),
    "updatedAt"   = date_trunc('milliseconds', "updatedAt"),
    "publishedAt" = date_trunc('milliseconds', "publishedAt")
WHERE "createdAt" <> date_trunc('milliseconds', "createdAt")
   OR "updatedAt" <> date_trunc('milliseconds', "updatedAt")
   OR "publishedAt" <> date_trunc('milliseconds', "publishedAt");

UPDATE sessions
SET "createdAt"      = date_trunc('milliseconds', "createdAt"),
    "lastActivityAt" = date_trunc('milliseconds', "lastActivityAt")
WHERE "createdAt" <> date_trunc('milliseconds', "createdAt")
   OR "lastActivityAt" <> date_trunc('milliseconds', "lastActivityAt");

UPDATE post_html_cache
SET "renderedAt" = date_trunc('milliseconds', "renderedAt")
WHERE "renderedAt" <> date_trunc('milliseconds', "renderedAt");
