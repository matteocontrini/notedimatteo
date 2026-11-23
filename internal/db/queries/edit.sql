-- Queries for the editor, which works on drafts and published posts alike.

-- name: PostForEdit :one
-- Returns a post with its tags, whether or not it's published.
SELECT p.id,
       p.number,
       p.category,
       COALESCE(p.title, '')::text AS title,
       COALESCE(p.slug, '')::text  AS slug,
       p."hideTitle"               AS hide_title,
       p.body,
       p."createdAt"               AS created_at,
       p."updatedAt"               AS updated_at,
       p."publishedAt"             AS published_at,
       ARRAY(
           SELECT pt."tagName" FROM posts_tags pt WHERE pt."postId" = p.id ORDER BY pt."position"
       )::text[]                   AS tags
FROM posts p
WHERE p.number = @number;

-- name: LockPost :one
-- Locks a post's row until the transaction ends, so two saves can't overlap.
SELECT id, body, "bodyRevision" AS body_revision, "publishedAt" AS published_at
FROM posts
WHERE number = @number
FOR UPDATE;

-- name: UpdatePost :exec
-- Saves the editor's fields. An empty title or slug becomes NULL.
UPDATE posts
SET number         = @number,
    category       = @category,
    title          = NULLIF(@title::text, ''),
    slug           = NULLIF(@slug::text, ''),
    "hideTitle"    = @hide_title,
    body           = @body,
    "bodyRevision" = @body_revision,
    "updatedAt"    = @updated_at::timestamptz,
    "publishedAt"  = sqlc.narg(published_at)
WHERE id = @id;

-- name: DeletePostTags :exec
DELETE FROM posts_tags
WHERE "postId" = @post_id;

-- name: CreateTags :exec
-- Adds the tags that don't exist yet.
INSERT INTO tags (name)
SELECT unnest(@names::text[])
ON CONFLICT DO NOTHING;

-- name: AddPostTags :exec
-- Adds the tags to the post and saves their order in position.
INSERT INTO posts_tags ("postId", "tagName", "position")
SELECT @post_id, t.name, (t.ordinality - 1)::int
FROM unnest(@names::text[]) WITH ORDINALITY AS t(name, ordinality);
