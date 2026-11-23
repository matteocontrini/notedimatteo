package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/matteocontrini/notedimatteo/internal/markdown"
)

// Post is a published post or a draft.
type Post struct {
	ID           int
	Number       int
	Category     string
	Title        string
	Slug         string
	HideTitle    bool
	Body         string
	BodyRevision int
	CreatedAt    time.Time
	UpdatedAt    *time.Time
	PublishedAt  *time.Time
	// Tags are in the order they were typed.
	Tags []string
	// CachedHTML is nil if the cache is missing, or was rendered from another
	// body revision or by another renderer version.
	CachedHTML *string
}

func newPost(row PostsRow) (Post, error) {
	tags, err := decodeTags(row.Tags)
	if err != nil {
		return Post{}, err
	}
	return Post{
		ID:           row.ID,
		Number:       row.Number,
		Category:     row.Category,
		Title:        row.Title,
		Slug:         row.Slug,
		HideTitle:    row.HideTitle,
		Body:         row.Body,
		BodyRevision: row.BodyRevision,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    timePointer(row.UpdatedAt),
		PublishedAt:  timePointer(row.PublishedAt),
		Tags:         tags,
		CachedHTML:   row.CachedHTML,
	}, nil
}

// Timestamp is the publication time, or the creation time for drafts.
func (p Post) Timestamp() time.Time {
	if p.PublishedAt != nil {
		return *p.PublishedAt
	}
	return p.CreatedAt
}

// PostFilter selects published posts. Zero fields don't filter.
type PostFilter struct {
	// Number is a pointer because 0 is a valid post number.
	Number   *int
	Category string
	// Tag matches a tag name exactly.
	Tag string
	// Search matches titles, bodies, and tag names containing it, ignoring case.
	Search string
	// From and Until select posts published at or after From and before Until.
	From  time.Time
	Until time.Time
}

// likeEscaper escapes LIKE's wildcards with the queries' escape character.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (f PostFilter) params() CountPostsParams {
	p := CountPostsParams{Number: f.Number}
	if f.Category != "" {
		p.Category = &f.Category
	}
	if f.Tag != "" {
		p.Tag = &f.Tag
	}
	if f.Search != "" {
		pattern := "%" + likeEscaper.Replace(f.Search) + "%"
		p.SearchPattern = &pattern
	}
	if !f.From.IsZero() {
		p.PublishedFrom = &Time{f.From}
	}
	if !f.Until.IsZero() {
		p.PublishedUntil = &Time{f.Until}
	}
	return p
}

// Posts returns published posts, newest first. A limit of 0 returns them all.
func (db *DB) Posts(ctx context.Context, filter PostFilter, limit, offset int) ([]Post, error) {
	p := filter.params()
	params := PostsParams{
		RendererVersion: markdown.Version,
		Number:          p.Number,
		Category:        p.Category,
		Tag:             p.Tag,
		SearchPattern:   p.SearchPattern,
		PublishedFrom:   p.PublishedFrom,
		PublishedUntil:  p.PublishedUntil,
		MaxRows:         limit,
		SkipRows:        offset,
	}
	if limit == 0 {
		params.MaxRows = -1
	}

	rows, err := db.queries.Posts(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}
	posts := make([]Post, len(rows))
	for i, row := range rows {
		if posts[i], err = newPost(row); err != nil {
			return nil, err
		}
	}
	return posts, nil
}

// CountPosts returns how many published posts match the filter.
func (db *DB) CountPosts(ctx context.Context, filter PostFilter) (int, error) {
	count, err := db.queries.CountPosts(ctx, filter.params())
	if err != nil {
		return 0, fmt.Errorf("count posts: %w", err)
	}
	return count, nil
}

// PostByNumber returns a published post, or nil if there's none.
func (db *DB) PostByNumber(ctx context.Context, number int) (*Post, error) {
	posts, err := db.Posts(ctx, PostFilter{Number: &number}, 1, 0)
	if err != nil || len(posts) == 0 {
		return nil, err
	}
	return &posts[0], nil
}

// SavePostHTML caches a post's rendered body for the current renderer version.
func (db *DB) SavePostHTML(ctx context.Context, postID, bodyRevision int, html string) error {
	err := db.queries.SavePostHTML(ctx, SavePostHTMLParams{
		PostID:          postID,
		RendererVersion: markdown.Version,
		BodyRevision:    bodyRevision,
		HTML:            html,
		RenderedAt:      currentTime(),
	})
	if err != nil {
		return fmt.Errorf("save post HTML: %w", err)
	}
	return nil
}
