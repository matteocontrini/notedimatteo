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
	ID          int       `db:"id"`
	Number      int       `db:"number"`
	Category    string    `db:"category"`
	Title       string    `db:"title"`
	Slug        string    `db:"slug"`
	HideTitle   bool      `db:"hideTitle"`
	Body        string    `db:"body"`
	PublishedAt time.Time `db:"publishedAt"`
	// CachedHTML is nil if the cache is missing or out of date.
	CachedHTML *string `db:"cachedHtml"`
}

// LatestPosts returns the most recently published posts, newest first.
func (db *DB) LatestPosts(ctx context.Context, limit int) ([]Post, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT p.id,
		       p.number,
		       p.category,
		       COALESCE(p.title, '') AS title,
		       COALESCE(p.slug, '')  AS slug,
		       p."hideTitle",
		       p.body,
		       p."publishedAt",
		       CASE
		           WHEN c."rendererVersion" = $1 AND c."bodyRevision" = p."bodyRevision" THEN c.html
		       END AS "cachedHtml"
		FROM posts p
		LEFT JOIN post_html_cache c ON c."postId" = p.id
		WHERE p."publishedAt" IS NOT NULL
		ORDER BY p."publishedAt" DESC, p.number DESC
		LIMIT $2
	`, RendererVersion, limit)
	if err != nil {
		return nil, fmt.Errorf("query latest posts: %w", err)
	}

	posts, err := pgx.CollectRows(rows, pgx.RowToStructByName[Post])
	if err != nil {
		return nil, fmt.Errorf("read latest posts: %w", err)
	}

	return posts, nil
}
