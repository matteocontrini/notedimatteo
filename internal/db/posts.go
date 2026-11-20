package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// RendererVersion matches MARKDOWN_RENDERER_VERSION in the SvelteKit app, so
// the cache written by that app stays valid until the Go renderer replaces it.
const RendererVersion = 1

type Post struct {
	ID          int        `db:"id"`
	Number      int        `db:"number"`
	Category    string     `db:"category"`
	Title       string     `db:"title"`
	Slug        string     `db:"slug"`
	HideTitle   bool       `db:"hideTitle"`
	Body        string     `db:"body"`
	CreatedAt   time.Time  `db:"createdAt"`
	PublishedAt *time.Time `db:"publishedAt"`
	Tags        []string   `db:"tags"`
	// CachedHTML is nil if the cache is missing or out of date.
	CachedHTML *string `db:"cachedHtml"`
}

// Timestamp is the publication time, or the creation time for drafts.
func (p Post) Timestamp() time.Time {
	if p.PublishedAt != nil {
		return *p.PublishedAt
	}
	return p.CreatedAt
}

// postColumns selects a Post from posts p joined with post_html_cache c.
// $1 is the renderer version.
//
// Tags have no ORDER BY, like the SvelteKit app: they come in whatever order
// Postgres returns them.
const postColumns = `
	p.id,
	p.number,
	p.category,
	COALESCE(p.title, '') AS title,
	COALESCE(p.slug, '')  AS slug,
	p."hideTitle",
	p.body,
	p."createdAt",
	p."publishedAt",
	ARRAY(
		SELECT pt."tagName" FROM posts_tags pt WHERE pt."postId" = p.id
	) AS tags,
	CASE
		WHEN c."rendererVersion" = $1 AND c."bodyRevision" = p."bodyRevision" THEN c.html
	END AS "cachedHtml"
`

// PublishedPosts returns a page of published posts, newest first.
func (db *DB) PublishedPosts(ctx context.Context, limit, offset int) ([]Post, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT `+postColumns+`
		FROM posts p
		LEFT JOIN post_html_cache c ON c."postId" = p.id
		WHERE p."publishedAt" IS NOT NULL
		ORDER BY p."publishedAt" DESC, p.number DESC
		LIMIT $2 OFFSET $3
	`, RendererVersion, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query published posts: %w", err)
	}

	posts, err := pgx.CollectRows(rows, pgx.RowToStructByName[Post])
	if err != nil {
		return nil, fmt.Errorf("read published posts: %w", err)
	}

	return posts, nil
}

func (db *DB) CountPublishedPosts(ctx context.Context) (int, error) {
	var count int
	err := db.pool.QueryRow(ctx, `SELECT COUNT(*) FROM posts WHERE "publishedAt" IS NOT NULL`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count published posts: %w", err)
	}
	return count, nil
}
