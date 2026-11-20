-- name: Categories :many
-- Categories of published posts, most used first.
SELECT category AS slug, COUNT(*)::int AS count
FROM posts
WHERE "publishedAt" IS NOT NULL
GROUP BY category
ORDER BY count DESC, slug;

-- name: Archive :many
-- Published posts per month, newest first, in the database session's time zone.
-- TODO: after the migration, group by month in Europe/Rome, like the dates
-- shown on posts, whatever the session's time zone is.
SELECT EXTRACT(YEAR FROM "publishedAt")::int  AS year,
       EXTRACT(MONTH FROM "publishedAt")::int AS month,
       COUNT(*)::int                          AS count
FROM posts
WHERE "publishedAt" IS NOT NULL
GROUP BY 1, 2
ORDER BY 1 DESC, 2 DESC;

-- name: Tags :many
-- Tags of published posts, most used first.
SELECT pt."tagName" AS name, COUNT(*)::int AS count
FROM posts_tags pt
JOIN posts p ON p.id = pt."postId"
WHERE p."publishedAt" IS NOT NULL
GROUP BY pt."tagName"
ORDER BY count DESC, name;
