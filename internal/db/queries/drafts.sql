-- A draft is a post with no publishedAt.

-- name: Drafts :many
-- Returns drafts, newest first. The columns match Posts, so Go can convert
-- each row to Post.
SELECT p.id,
       p.number,
       p.category,
       p.title,
       p.slug,
       p."hideTitle"               AS hide_title,
       p.body,
       p."bodyRevision"            AS body_revision,
       p."createdAt"               AS created_at,
       p."updatedAt"               AS updated_at,
       p."publishedAt"             AS published_at,
       ARRAY(
           SELECT pt."tagName" FROM posts_tags pt WHERE pt."postId" = p.id ORDER BY pt."position"
       )::text[]                   AS tags,
       c.html                      AS cached_html
FROM posts p
LEFT JOIN post_html_cache c
       ON c."postId" = p.id
      AND c."rendererVersion" = @renderer_version::int
      AND c."bodyRevision" = p."bodyRevision"
WHERE p."publishedAt" IS NULL
ORDER BY p."createdAt" DESC;

-- name: CountDrafts :one
SELECT COUNT(*)::int
FROM posts
WHERE "publishedAt" IS NULL;

-- name: CreateDraft :one
-- Creates an empty note with the next post number.
INSERT INTO posts (number, category, "hideTitle", body, "createdAt")
SELECT COALESCE(MAX(number), 0) + 1, 'note', FALSE, '', @created_at::timestamptz
FROM posts
RETURNING number;

-- name: DeleteDraft :execrows
-- Postgres also deletes the post's tags and cached HTML (ON DELETE CASCADE).
DELETE FROM posts
WHERE number = @number
  AND "publishedAt" IS NULL;
