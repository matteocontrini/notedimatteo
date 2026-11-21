-- Posts and CountPosts take the same optional filters: a NULL parameter
-- doesn't filter. Keep their WHERE clauses identical.

-- name: Posts :many
-- Published posts, newest first. Tags have no ORDER BY: they come in whatever
-- order Postgres returns them.
SELECT p.id,
       p.number,
       p.category,
       COALESCE(p.title, '')::text AS title,
       COALESCE(p.slug, '')::text  AS slug,
       p."hideTitle"               AS hide_title,
       p.body,
       p."bodyRevision"            AS body_revision,
       p."createdAt"               AS created_at,
       p."updatedAt"               AS updated_at,
       p."publishedAt"             AS published_at,
       ARRAY(
           SELECT pt."tagName" FROM posts_tags pt WHERE pt."postId" = p.id
       )::text[]                   AS tags,
       c.html                      AS cached_html
FROM posts p
-- No cached HTML if it's from another body revision or renderer version
LEFT JOIN post_html_cache c
       ON c."postId" = p.id
      AND c."rendererVersion" = @renderer_version::int
      AND c."bodyRevision" = p."bodyRevision"
WHERE p."publishedAt" IS NOT NULL
  AND (sqlc.narg(number)::int IS NULL OR p.number = sqlc.narg(number))
  AND (sqlc.narg(category)::text IS NULL OR p.category = sqlc.narg(category))
  AND (sqlc.narg(tag)::text IS NULL OR EXISTS (
          SELECT 1 FROM posts_tags pt WHERE pt."postId" = p.id AND pt."tagName" = sqlc.narg(tag)))
  AND (sqlc.narg(search_pattern)::text IS NULL
          OR p.title ILIKE sqlc.narg(search_pattern)
          OR p.body ILIKE sqlc.narg(search_pattern)
          OR EXISTS (SELECT 1 FROM posts_tags pt WHERE pt."postId" = p.id AND pt."tagName" ILIKE sqlc.narg(search_pattern)))
  AND (sqlc.narg(published_from)::timestamptz IS NULL OR p."publishedAt" >= sqlc.narg(published_from))
  AND (sqlc.narg(published_until)::timestamptz IS NULL OR p."publishedAt" < sqlc.narg(published_until))
ORDER BY p."publishedAt" DESC, p.number DESC
-- LIMIT NULL returns every row
LIMIT sqlc.narg(max_rows)::int OFFSET @skip_rows::int;

-- name: CountPosts :one
SELECT COUNT(*)::int
FROM posts p
WHERE p."publishedAt" IS NOT NULL
  AND (sqlc.narg(number)::int IS NULL OR p.number = sqlc.narg(number))
  AND (sqlc.narg(category)::text IS NULL OR p.category = sqlc.narg(category))
  AND (sqlc.narg(tag)::text IS NULL OR EXISTS (
          SELECT 1 FROM posts_tags pt WHERE pt."postId" = p.id AND pt."tagName" = sqlc.narg(tag)))
  AND (sqlc.narg(search_pattern)::text IS NULL
          OR p.title ILIKE sqlc.narg(search_pattern)
          OR p.body ILIKE sqlc.narg(search_pattern)
          OR EXISTS (SELECT 1 FROM posts_tags pt WHERE pt."postId" = p.id AND pt."tagName" ILIKE sqlc.narg(search_pattern)))
  AND (sqlc.narg(published_from)::timestamptz IS NULL OR p."publishedAt" >= sqlc.narg(published_from))
  AND (sqlc.narg(published_until)::timestamptz IS NULL OR p."publishedAt" < sqlc.narg(published_until));

-- name: SavePostHTML :exec
-- Caches a post's rendered body.
INSERT INTO post_html_cache ("postId", "rendererVersion", "bodyRevision", html)
VALUES (@post_id, @renderer_version, @body_revision, @html)
ON CONFLICT ("postId") DO UPDATE
SET "rendererVersion" = EXCLUDED."rendererVersion",
    "bodyRevision"    = EXCLUDED."bodyRevision",
    html              = EXCLUDED.html,
    "renderedAt"      = CURRENT_TIMESTAMP;
