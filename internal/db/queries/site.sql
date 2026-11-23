-- name: Categories :many
-- Categories of published posts, most used first.
SELECT category AS slug, COUNT(*)::int AS count
FROM posts
WHERE published_at IS NOT NULL
GROUP BY category
ORDER BY count DESC, slug;

-- name: PublishedTimes :many
-- Returns the publication times of published posts, newest first.
SELECT published_at
FROM posts
WHERE published_at IS NOT NULL
ORDER BY published_at DESC;

-- name: Tags :many
-- Tags of published posts, most used first.
SELECT pt.tag_name AS name, COUNT(*)::int AS count
FROM posts_tags pt
JOIN posts p ON p.id = pt.post_id
WHERE p.published_at IS NOT NULL
GROUP BY pt.tag_name
ORDER BY count DESC, name;
