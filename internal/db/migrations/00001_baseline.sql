-- Create the schema in SQLite. It's the PostgreSQL schema after its migration 00007,
-- plus indexes on posts.published_at, posts_tags.tag_name, and sessions.token.
--
-- STRICT tables reject values of the wrong type. They have no time or boolean
-- types, so:
-- - Times are TEXT in UTC with milliseconds, like 2026-11-23 10:00:00.000, the
--   format of SQLite's date functions. All times have the same width, so comparing
--   and sorting the text compares and sorts the times. A CHECK rejects any other
--   format. TestTimeColumns checks that every *_at column has it.
-- - Booleans are INTEGER, with a CHECK for 0 and 1.
--
-- sqlc.yaml maps these columns to Go types.

-- +goose Up
CREATE TABLE posts (
    id            INTEGER PRIMARY KEY,
    number        INTEGER NOT NULL UNIQUE,
    category      TEXT    NOT NULL,
    created_at    TEXT    NOT NULL CHECK (created_at IS strftime('%Y-%m-%d %H:%M:%f', created_at)),
    updated_at    TEXT             CHECK (updated_at IS strftime('%Y-%m-%d %H:%M:%f', updated_at)),
    published_at  TEXT             CHECK (published_at IS strftime('%Y-%m-%d %H:%M:%f', published_at)),
    title         TEXT    NOT NULL DEFAULT '',
    slug          TEXT    NOT NULL DEFAULT '',
    hide_title    INTEGER NOT NULL CHECK (hide_title IN (0, 1)),
    body          TEXT    NOT NULL,
    body_revision INTEGER NOT NULL DEFAULT 1
) STRICT;

CREATE INDEX posts_published_at ON posts (published_at);

CREATE TABLE tags (
    name TEXT PRIMARY KEY
) STRICT, WITHOUT ROWID;

CREATE TABLE posts_tags (
    post_id  INTEGER NOT NULL REFERENCES posts (id) ON DELETE CASCADE ON UPDATE CASCADE,
    tag_name TEXT    NOT NULL REFERENCES tags (name) ON DELETE CASCADE ON UPDATE CASCADE,
    position INTEGER NOT NULL,
    PRIMARY KEY (post_id, tag_name)
) STRICT, WITHOUT ROWID;

CREATE INDEX posts_tags_tag_name ON posts_tags (tag_name);

CREATE TABLE sessions (
    id               INTEGER PRIMARY KEY,
    token            TEXT NOT NULL UNIQUE,
    created_at       TEXT NOT NULL CHECK (created_at IS strftime('%Y-%m-%d %H:%M:%f', created_at)),
    last_activity_at TEXT NOT NULL CHECK (last_activity_at IS strftime('%Y-%m-%d %H:%M:%f', last_activity_at))
) STRICT;

CREATE TABLE post_html_cache (
    post_id          INTEGER PRIMARY KEY REFERENCES posts (id) ON DELETE CASCADE ON UPDATE CASCADE,
    renderer_version INTEGER NOT NULL,
    body_revision    INTEGER NOT NULL,
    html             TEXT    NOT NULL,
    rendered_at      TEXT    NOT NULL CHECK (rendered_at IS strftime('%Y-%m-%d %H:%M:%f', rendered_at))
) STRICT;
