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
       ARRAY(
           SELECT pt.tag_name FROM posts_tags pt WHERE pt.post_id = p.id ORDER BY pt.position
       )::text[] AS tags
FROM posts p
WHERE p.number = @number;

-- name: LockPost :one
-- Locks a post's row until the transaction ends, so two saves can't overlap.
SELECT id, body, body_revision, published_at
FROM posts
WHERE number = @number
FOR UPDATE;

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
    updated_at    = @updated_at::timestamptz,
    published_at  = sqlc.narg(published_at)
WHERE id = @id;

-- name: DeletePostTags :exec
DELETE FROM posts_tags
WHERE post_id = @post_id;

-- name: CreateTags :exec
-- Adds the tags that don't exist yet.
INSERT INTO tags (name)
SELECT unnest(@names::text[])
ON CONFLICT DO NOTHING;

-- name: AddPostTags :exec
-- Adds the tags to the post and saves their order in position.
INSERT INTO posts_tags (post_id, tag_name, position)
SELECT @post_id, t.name, (t.ordinality - 1)::int
FROM unnest(@names::text[]) WITH ORDINALITY AS t(name, ordinality);
