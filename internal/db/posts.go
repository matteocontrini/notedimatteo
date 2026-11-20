package db

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
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
	UpdatedAt    *time.Time `db:"updatedAt"`
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
	p."updatedAt",
	p."publishedAt",
	ARRAY(
		SELECT pt."tagName" FROM posts_tags pt WHERE pt."postId" = p.id
	) AS tags,
	CASE
		WHEN c."rendererVersion" = $1 AND c."bodyRevision" = p."bodyRevision" THEN c.html
	END AS "cachedHtml"
`

// PostFilter selects published posts. Empty fields don't filter.
type PostFilter struct {
	Category string
	// Tag matches a tag name exactly.
	Tag string
	// Search matches titles, bodies, and tag names containing it, ignoring case.
	Search string
	// From and Until select posts published at or after From and before Until.
	From  time.Time
	Until time.Time
}

// likeEscaper escapes LIKE's wildcards, with ILIKE's default escape character.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// where returns the WHERE clause, appending its arguments to args.
func (f PostFilter) where(args []any) (string, []any) {
	conditions := []string{`p."publishedAt" IS NOT NULL`}
	arg := func(value any) string {
		args = append(args, value)
		return "$" + strconv.Itoa(len(args))
	}

	if f.Category != "" {
		conditions = append(conditions, `p.category = `+arg(f.Category))
	}
	if f.Tag != "" {
		conditions = append(conditions,
			`EXISTS (SELECT 1 FROM posts_tags pt WHERE pt."postId" = p.id AND pt."tagName" = `+arg(f.Tag)+`)`)
	}
	if f.Search != "" {
		pattern := arg(likeEscaper.Replace(f.Search))
		contains := `'%' || ` + pattern + ` || '%'`
		conditions = append(conditions, `(p.title ILIKE `+contains+
			` OR p.body ILIKE `+contains+
			` OR EXISTS (SELECT 1 FROM posts_tags pt WHERE pt."postId" = p.id AND pt."tagName" ILIKE `+contains+`))`)
	}
	if !f.From.IsZero() {
		conditions = append(conditions, `p."publishedAt" >= `+arg(f.From))
	}
	if !f.Until.IsZero() {
		conditions = append(conditions, `p."publishedAt" < `+arg(f.Until))
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

// Posts returns published posts, newest first. A limit of 0 returns them all.
func (db *DB) Posts(ctx context.Context, filter PostFilter, limit, offset int) ([]Post, error) {
	where, args := filter.where([]any{markdown.Version})
	sql := `
		SELECT ` + postColumns + `
		FROM posts p
		LEFT JOIN post_html_cache c ON c."postId" = p.id
		` + where + `
		ORDER BY p."publishedAt" DESC, p.number DESC`
	if limit > 0 {
		args = append(args, limit, offset)
		sql += ` LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))
	}

	rows, err := db.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}

	posts, err := pgx.CollectRows(rows, pgx.RowToStructByName[Post])
	if err != nil {
		return nil, fmt.Errorf("read posts: %w", err)
	}

	return posts, nil
}

// CountPosts returns how many published posts match the filter.
func (db *DB) CountPosts(ctx context.Context, filter PostFilter) (int, error) {
	where, args := filter.where(nil)

	var count int
	if err := db.pool.QueryRow(ctx, `SELECT COUNT(*) FROM posts p `+where, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count posts: %w", err)
	}
	return count, nil
}

// PostByNumber returns a post, published or not, or nil if there's none.
func (db *DB) PostByNumber(ctx context.Context, number int) (*Post, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT `+postColumns+`
		FROM posts p
		LEFT JOIN post_html_cache c ON c."postId" = p.id
		WHERE p.number = $2
	`, markdown.Version, number)
	if err != nil {
		return nil, fmt.Errorf("query post %d: %w", number, err)
	}

	post, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[Post])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read post %d: %w", number, err)
	}
	return post, nil
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
