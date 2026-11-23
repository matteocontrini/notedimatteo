-- Rename the camelCase columns from the Prisma schema to snake_case, so queries
-- don't need quotes and aliases. Constraint and index names stay.

-- +goose Up
ALTER TABLE posts RENAME COLUMN "createdAt" TO created_at;
ALTER TABLE posts RENAME COLUMN "updatedAt" TO updated_at;
ALTER TABLE posts RENAME COLUMN "publishedAt" TO published_at;
ALTER TABLE posts RENAME COLUMN "hideTitle" TO hide_title;
ALTER TABLE posts RENAME COLUMN "bodyRevision" TO body_revision;

ALTER TABLE posts_tags RENAME COLUMN "postId" TO post_id;
ALTER TABLE posts_tags RENAME COLUMN "tagName" TO tag_name;

ALTER TABLE sessions RENAME COLUMN "createdAt" TO created_at;
ALTER TABLE sessions RENAME COLUMN "lastActivityAt" TO last_activity_at;

ALTER TABLE post_html_cache RENAME COLUMN "postId" TO post_id;
ALTER TABLE post_html_cache RENAME COLUMN "rendererVersion" TO renderer_version;
ALTER TABLE post_html_cache RENAME COLUMN "bodyRevision" TO body_revision;
ALTER TABLE post_html_cache RENAME COLUMN "renderedAt" TO rendered_at;
