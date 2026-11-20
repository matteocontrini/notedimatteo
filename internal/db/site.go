package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type CategoryCount struct {
	Slug  string `db:"slug"`
	Count int    `db:"count"`
}

type MonthCount struct {
	Year  int `db:"year"`
	Month int `db:"month"`
	Count int `db:"count"`
}

type TagCount struct {
	Name  string `db:"name"`
	Count int    `db:"count"`
}

// Categories returns the categories of published posts, most used first.
func (db *DB) Categories(ctx context.Context) ([]CategoryCount, error) {
	return query[CategoryCount](ctx, db, "categories", `
		SELECT category AS slug, COUNT(*) AS count
		FROM posts
		WHERE "publishedAt" IS NOT NULL
		GROUP BY category
		ORDER BY count DESC, slug
	`)
}

// Archive returns the number of published posts per month, newest first.
// Months follow the database session's time zone, like the SvelteKit app.
func (db *DB) Archive(ctx context.Context) ([]MonthCount, error) {
	return query[MonthCount](ctx, db, "archive", `
		SELECT EXTRACT(YEAR FROM "publishedAt")::int  AS year,
		       EXTRACT(MONTH FROM "publishedAt")::int AS month,
		       COUNT(*)                               AS count
		FROM posts
		WHERE "publishedAt" IS NOT NULL
		GROUP BY 1, 2
		ORDER BY 1 DESC, 2 DESC
	`)
}

// Tags returns the tags of published posts, most used first.
func (db *DB) Tags(ctx context.Context) ([]TagCount, error) {
	return query[TagCount](ctx, db, "tags", `
		SELECT pt."tagName" AS name, COUNT(*) AS count
		FROM posts_tags pt
		JOIN posts p ON p.id = pt."postId"
		WHERE p."publishedAt" IS NOT NULL
		GROUP BY pt."tagName"
		ORDER BY count DESC, name
	`)
}

func query[T any](ctx context.Context, db *DB, name, sql string, args ...any) ([]T, error) {
	rows, err := db.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query %s: %w", name, err)
	}

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[T])
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}

	return items, nil
}
