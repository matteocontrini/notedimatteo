-- This migration creates the schema that the previous site Prisma migrations created.
-- Every statement skips what exists, so on an existing database it changes
-- nothing, and goose only records it as applied.

-- +goose Up
CREATE TABLE IF NOT EXISTS posts (
    "id" SERIAL NOT NULL,
    "number" INTEGER NOT NULL,
    "category" TEXT NOT NULL,
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    "publishedAt" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    "title" TEXT,
    "slug" TEXT,
    "hideTitle" BOOLEAN NOT NULL,
    "body" TEXT NOT NULL,
    "bodyRevision" INTEGER NOT NULL DEFAULT 1,

    CONSTRAINT "posts_pkey" PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX IF NOT EXISTS "posts_number_key" ON posts ("number");

CREATE TABLE IF NOT EXISTS tags (
    "name" TEXT NOT NULL,

    CONSTRAINT "tags_pkey" PRIMARY KEY ("name")
);

CREATE TABLE IF NOT EXISTS posts_tags (
    "postId" INTEGER NOT NULL,
    "tagName" TEXT NOT NULL,

    CONSTRAINT "posts_tags_pkey" PRIMARY KEY ("postId", "tagName"),
    CONSTRAINT "posts_tags_postId_fkey" FOREIGN KEY ("postId") REFERENCES posts ("id") ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT "posts_tags_tagName_fkey" FOREIGN KEY ("tagName") REFERENCES tags ("name") ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS sessions (
    "id" SERIAL NOT NULL,
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "lastActivityAt" TIMESTAMPTZ NOT NULL,
    "token" TEXT NOT NULL,

    CONSTRAINT "sessions_pkey" PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS post_html_cache (
    "postId" INTEGER NOT NULL,
    "rendererVersion" INTEGER NOT NULL,
    "bodyRevision" INTEGER NOT NULL,
    "html" TEXT NOT NULL,
    "renderedAt" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "post_html_cache_pkey" PRIMARY KEY ("postId"),
    CONSTRAINT "post_html_cache_postId_fkey" FOREIGN KEY ("postId") REFERENCES posts ("id") ON DELETE CASCADE ON UPDATE CASCADE
);

-- +goose Down
-- Rolling back the baseline would drop every table, so it does nothing.
