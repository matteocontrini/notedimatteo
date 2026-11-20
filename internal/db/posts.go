package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/matteocontrini/notedimatteo/internal/markdown"
)

type Post struct {
	ID        int    `db:"id"`
	Number    int    `db:"number"`
	Category  string `db:"category"`
	Title     string `db:"title"`
	Slug      string `db:"slug"`
	HideTitle bool   `db:"hideTitle"`
	Body      string `db:"body"`
	// BodyRevision increases whenever the body changes, and is stored with cached HTML.
	BodyRevision int        `db:"bodyRevision"`
	CreatedAt    time.Time  `db:"createdAt"`
	PublishedAt  *time.Time `db:"publishedAt"`
	Tags         []string   `db:"tags"`
	// CachedHTML is nil if the cache is missing, or was rendered from an older
	// body revision or by another renderer version.
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
	p."bodyRevision",
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
	`, markdown.Version, limit, offset)
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

// SavePostHTML caches a post's rendered body for the current renderer version.
func (db *DB) SavePostHTML(ctx context.Context, postID, bodyRevision int, html string) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO post_html_cache ("postId", "rendererVersion", "bodyRevision", html)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT ("postId") DO UPDATE
		SET "rendererVersion" = EXCLUDED."rendererVersion",
		    "bodyRevision"    = EXCLUDED."bodyRevision",
		    html              = EXCLUDED.html,
		    "renderedAt"      = CURRENT_TIMESTAMP
	`, postID, markdown.Version, bodyRevision, html)
	if err != nil {
		return fmt.Errorf("save post HTML: %w", err)
	}
	return nil
}
