-- A post's tags keep the order they were typed in, which position records.
-- Saves have always inserted a post's tags in that order, so the rows' physical
-- order (ctid) is the best record of it: rows usually stay where they were
-- inserted, but Postgres can reuse space freed by deleted rows.

-- +goose Up
ALTER TABLE posts_tags ADD COLUMN "position" INTEGER;

UPDATE posts_tags pt
SET "position" = o.position
FROM (
    SELECT ctid, row_number() OVER (PARTITION BY "postId" ORDER BY ctid) - 1 AS position
    FROM posts_tags
) o
WHERE pt.ctid = o.ctid;

ALTER TABLE posts_tags ALTER COLUMN "position" SET NOT NULL;
