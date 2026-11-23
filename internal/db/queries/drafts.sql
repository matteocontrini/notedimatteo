-- A draft is a post with no published_at.

-- name: Drafts :many
-- Returns drafts, newest first. The columns match Posts, so Go can convert
-- each row to Post.
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
LEFT JOIN post_html_cache c
       ON c.post_id = p.id
      AND c.renderer_version = @renderer_version
      AND c.body_revision = p.body_revision
WHERE p.published_at IS NULL
ORDER BY p.created_at DESC;

-- name: CountDrafts :one
SELECT COUNT(*)
FROM posts
WHERE published_at IS NULL;

-- name: CreateDraft :one
-- Creates an empty note with the next post number.
INSERT INTO posts (number, category, hide_title, body, created_at)
SELECT COALESCE(MAX(number), 0) + 1, 'note', FALSE, '', @created_at
FROM posts
RETURNING number;

-- name: DeleteDraft :execrows
-- The database also deletes the post's tags and cached HTML (ON DELETE CASCADE).
DELETE FROM posts
WHERE number = @number
  AND published_at IS NULL;
