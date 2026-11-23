-- Posts and CountPosts take the same optional filters: a NULL parameter
-- doesn't filter. Keep their WHERE clauses identical.

-- name: Posts :many
-- Returns published posts, newest first, with tags in typed order.
SELECT p.id,
       p.number,
       p.category,
       p.title,
       p.slug,
       p.hide_title,
       p.body,
       p.body_revision,
       p.created_at,
       p.updated_at,
       p.published_at,
       ARRAY(
           SELECT pt.tag_name FROM posts_tags pt WHERE pt.post_id = p.id ORDER BY pt.position
       )::text[] AS tags,
       c.html AS cached_html
FROM posts p
-- No cached HTML if it's from another body revision or renderer version
LEFT JOIN post_html_cache c
       ON c.post_id = p.id
      AND c.renderer_version = @renderer_version::int
      AND c.body_revision = p.body_revision
WHERE p.published_at IS NOT NULL
  AND (sqlc.narg(number)::int IS NULL OR p.number = sqlc.narg(number))
  AND (sqlc.narg(category)::text IS NULL OR p.category = sqlc.narg(category))
  AND (sqlc.narg(tag)::text IS NULL OR EXISTS (
          SELECT 1 FROM posts_tags pt WHERE pt.post_id = p.id AND pt.tag_name = sqlc.narg(tag)))
  AND (sqlc.narg(search_pattern)::text IS NULL
          OR p.title ILIKE sqlc.narg(search_pattern)
          OR p.body ILIKE sqlc.narg(search_pattern)
          OR EXISTS (SELECT 1 FROM posts_tags pt WHERE pt.post_id = p.id AND pt.tag_name ILIKE sqlc.narg(search_pattern)))
  AND (sqlc.narg(published_from)::timestamptz IS NULL OR p.published_at >= sqlc.narg(published_from))
  AND (sqlc.narg(published_until)::timestamptz IS NULL OR p.published_at < sqlc.narg(published_until))
ORDER BY p.published_at DESC, p.number DESC
-- LIMIT NULL returns every row
LIMIT sqlc.narg(max_rows)::int OFFSET @skip_rows::int;

-- name: CountPosts :one
SELECT COUNT(*)::int
FROM posts p
WHERE p.published_at IS NOT NULL
  AND (sqlc.narg(number)::int IS NULL OR p.number = sqlc.narg(number))
  AND (sqlc.narg(category)::text IS NULL OR p.category = sqlc.narg(category))
  AND (sqlc.narg(tag)::text IS NULL OR EXISTS (
          SELECT 1 FROM posts_tags pt WHERE pt.post_id = p.id AND pt.tag_name = sqlc.narg(tag)))
  AND (sqlc.narg(search_pattern)::text IS NULL
          OR p.title ILIKE sqlc.narg(search_pattern)
          OR p.body ILIKE sqlc.narg(search_pattern)
          OR EXISTS (SELECT 1 FROM posts_tags pt WHERE pt.post_id = p.id AND pt.tag_name ILIKE sqlc.narg(search_pattern)))
  AND (sqlc.narg(published_from)::timestamptz IS NULL OR p.published_at >= sqlc.narg(published_from))
  AND (sqlc.narg(published_until)::timestamptz IS NULL OR p.published_at < sqlc.narg(published_until));

-- name: SavePostHTML :exec
-- Caches a post's rendered body.
INSERT INTO post_html_cache (post_id, renderer_version, body_revision, html, rendered_at)
VALUES (@post_id, @renderer_version, @body_revision, @html, @rendered_at)
ON CONFLICT (post_id) DO UPDATE
SET renderer_version = EXCLUDED.renderer_version,
    body_revision    = EXCLUDED.body_revision,
    html             = EXCLUDED.html,
    rendered_at      = EXCLUDED.rendered_at;
