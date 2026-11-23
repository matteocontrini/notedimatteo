-- Posts and CountPosts take the same optional filters: a NULL parameter
-- doesn't filter. Keep their WHERE clauses identical.
--
-- Each filter compares the parameter with a column before testing it for NULL:
-- sqlc types a parameter from its first use, and the column gives it a type.
--
-- LIKE ignores case for every letter (the unicode extension, registered by
-- Open). sqlc can't parse a LIKE with ESCAPE after an OR without parentheses.

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
       CAST((SELECT json_group_array(tag_name) FROM (
           SELECT pt.tag_name FROM posts_tags pt WHERE pt.post_id = p.id ORDER BY pt.position
       )) AS TEXT) AS tags,
       c.html AS cached_html
FROM posts p
-- No cached HTML if it's from another body revision or renderer version
LEFT JOIN post_html_cache c
       ON c.post_id = p.id
      AND c.renderer_version = @renderer_version
      AND c.body_revision = p.body_revision
WHERE p.published_at IS NOT NULL
  AND (p.number = sqlc.narg(number) OR sqlc.narg(number) IS NULL)
  AND (p.category = sqlc.narg(category) OR sqlc.narg(category) IS NULL)
  AND (EXISTS (SELECT 1 FROM posts_tags pt WHERE pt.post_id = p.id AND pt.tag_name = sqlc.narg(tag))
          OR sqlc.narg(tag) IS NULL)
  AND ((p.title LIKE sqlc.narg(search_pattern) ESCAPE '\')
          OR (p.body LIKE sqlc.narg(search_pattern) ESCAPE '\')
          OR EXISTS (SELECT 1 FROM posts_tags pt WHERE pt.post_id = p.id
                     AND (pt.tag_name LIKE sqlc.narg(search_pattern) ESCAPE '\'))
          OR sqlc.narg(search_pattern) IS NULL)
  AND (p.published_at >= sqlc.narg(published_from) OR sqlc.narg(published_from) IS NULL)
  AND (p.published_at < sqlc.narg(published_until) OR sqlc.narg(published_until) IS NULL)
ORDER BY p.published_at DESC, p.number DESC
-- LIMIT -1 returns every row
LIMIT @max_rows OFFSET @skip_rows;

-- name: CountPosts :one
SELECT COUNT(*)
FROM posts p
WHERE p.published_at IS NOT NULL
  AND (p.number = sqlc.narg(number) OR sqlc.narg(number) IS NULL)
  AND (p.category = sqlc.narg(category) OR sqlc.narg(category) IS NULL)
  AND (EXISTS (SELECT 1 FROM posts_tags pt WHERE pt.post_id = p.id AND pt.tag_name = sqlc.narg(tag))
          OR sqlc.narg(tag) IS NULL)
  AND ((p.title LIKE sqlc.narg(search_pattern) ESCAPE '\')
          OR (p.body LIKE sqlc.narg(search_pattern) ESCAPE '\')
          OR EXISTS (SELECT 1 FROM posts_tags pt WHERE pt.post_id = p.id
                     AND (pt.tag_name LIKE sqlc.narg(search_pattern) ESCAPE '\'))
          OR sqlc.narg(search_pattern) IS NULL)
  AND (p.published_at >= sqlc.narg(published_from) OR sqlc.narg(published_from) IS NULL)
  AND (p.published_at < sqlc.narg(published_until) OR sqlc.narg(published_until) IS NULL);

-- name: SavePostHTML :exec
-- Caches a post's rendered body.
INSERT INTO post_html_cache (post_id, renderer_version, body_revision, html, rendered_at)
VALUES (@post_id, @renderer_version, @body_revision, @html, @rendered_at)
ON CONFLICT (post_id) DO UPDATE
SET renderer_version = excluded.renderer_version,
    body_revision    = excluded.body_revision,
    html             = excluded.html,
    rendered_at      = excluded.rendered_at;
