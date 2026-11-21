package db

import (
	"context"
	"fmt"

	"github.com/matteocontrini/notedimatteo/internal/markdown"
)

// Drafts returns unpublished posts, newest first.
func (db *DB) Drafts(ctx context.Context) ([]Post, error) {
	rows, err := db.queries.Drafts(ctx, markdown.Version)
	if err != nil {
		return nil, fmt.Errorf("query drafts: %w", err)
	}
	posts := make([]Post, len(rows))
	for i, row := range rows {
		posts[i] = Post(row)
	}
	return posts, nil
}

func (db *DB) CountDrafts(ctx context.Context) (int, error) {
	count, err := db.queries.CountDrafts(ctx)
	if err != nil {
		return 0, fmt.Errorf("count drafts: %w", err)
	}
	return count, nil
}

// CreateDraft creates an empty note with the next post number, and returns
// that number.
func (db *DB) CreateDraft(ctx context.Context) (int, error) {
	number, err := db.queries.CreateDraft(ctx)
	if err != nil {
		return 0, fmt.Errorf("create draft: %w", err)
	}
	return number, nil
}

// DeleteDraft deletes an unpublished post and reports whether it found one.
// It never deletes a published post.
func (db *DB) DeleteDraft(ctx context.Context, number int) (bool, error) {
	deleted, err := db.queries.DeleteDraft(ctx, number)
	if err != nil {
		return false, fmt.Errorf("delete draft: %w", err)
	}
	return deleted > 0, nil
}
