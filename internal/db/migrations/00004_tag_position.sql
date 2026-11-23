-- Add a position column to keep each post's tags in the order they were typed.
-- Fill it from the physical row order (ctid), because saves have always inserted
-- tags in typed order. A few posts may get a different order, where Postgres
-- reused space freed by deleted rows.

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
