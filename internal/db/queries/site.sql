-- name: Categories :many
-- Categories of published posts, most used first.
SELECT category AS slug, COUNT(*)::int AS count
FROM posts
WHERE "publishedAt" IS NOT NULL
GROUP BY category
ORDER BY count DESC, slug;

-- name: PublishedTimes :many
-- Returns the publication times of published posts, newest first.
SELECT "publishedAt"
FROM posts
WHERE "publishedAt" IS NOT NULL
ORDER BY "publishedAt" DESC;

-- name: Tags :many
-- Tags of published posts, most used first.
SELECT pt."tagName" AS name, COUNT(*)::int AS count
FROM posts_tags pt
JOIN posts p ON p.id = pt."postId"
WHERE p."publishedAt" IS NOT NULL
GROUP BY pt."tagName"
ORDER BY count DESC, name;
