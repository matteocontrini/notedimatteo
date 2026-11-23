-- Queries for the editor, which works on drafts and published posts alike.

-- name: PostForEdit :one
-- Returns a post with its tags, whether or not it's published.
SELECT p.id,
       p.number,
       p.category,
       p.title,
       p.slug,
       p.hide_title,
       p.body,
       p.created_at,
       p.updated_at,
       p.published_at,
       CAST((SELECT json_group_array(tag_name) FROM (
           SELECT pt.tag_name FROM posts_tags pt WHERE pt.post_id = p.id ORDER BY pt.position
       )) AS TEXT) AS tags
FROM posts p
WHERE p.number = @number;

-- name: PostForSave :one
-- Returns what a save changes. The save's transaction holds the write lock
-- from its start, so two saves can't overlap.
SELECT id, body, body_revision, published_at
FROM posts
WHERE number = @number;

-- name: UpdatePost :exec
-- Saves the editor's fields.
UPDATE posts
SET number        = @number,
    category      = @category,
    title         = @title,
    slug          = @slug,
    hide_title    = @hide_title,
    body          = @body,
    body_revision = @body_revision,
    updated_at    = @updated_at,
    published_at  = sqlc.narg(published_at)
WHERE id = @id;

-- name: DeletePostTags :exec
DELETE FROM posts_tags
WHERE post_id = @post_id;

-- name: CreateTags :exec
-- Adds the tags that don't exist yet. names is a JSON array of strings.
-- WHERE true lets the parser tell ON CONFLICT from a join constraint.
INSERT INTO tags (name)
SELECT value FROM json_each(CAST(@names AS TEXT))
WHERE true
ON CONFLICT DO NOTHING;

-- name: AddPostTags :exec
-- Adds the tags to the post and saves their order in position. names is a JSON
-- array of strings, and json_each's key is each name's index.
INSERT INTO posts_tags (post_id, tag_name, position)
SELECT @post_id, value, key
FROM json_each(CAST(@names AS TEXT));
