package db

import (
	"context"
	"fmt"
)

type (
	CategoryCount = CategoriesRow
	MonthCount    = ArchiveRow
	TagCount      = TagsRow
)

// Categories returns the categories of published posts, most used first.
func (db *DB) Categories(ctx context.Context) ([]CategoryCount, error) {
	categories, err := db.queries.Categories(ctx)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	return categories, nil
}

// Archive returns the number of published posts per month, newest first.
func (db *DB) Archive(ctx context.Context) ([]MonthCount, error) {
	months, err := db.queries.Archive(ctx)
	if err != nil {
		return nil, fmt.Errorf("query archive: %w", err)
	}
	return months, nil
}

// Tags returns the tags of published posts, most used first.
func (db *DB) Tags(ctx context.Context) ([]TagCount, error) {
	tags, err := db.queries.Tags(ctx)
	if err != nil {
		return nil, fmt.Errorf("query tags: %w", err)
	}
	return tags, nil
}
