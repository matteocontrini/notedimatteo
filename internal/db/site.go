package db

import (
	"context"
	"fmt"
	"time"
)

type (
	CategoryCount = CategoriesRow
	TagCount      = TagsRow
)

// MonthCount is the number of posts published in a month.
type MonthCount struct {
	Year  int
	Month int
	Count int
}

// Categories returns the categories of published posts, most used first.
func (db *DB) Categories(ctx context.Context) ([]CategoryCount, error) {
	categories, err := db.queries.Categories(ctx)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	return categories, nil
}

// Archive returns the number of published posts per month in loc, newest first.
func (db *DB) Archive(ctx context.Context, loc *time.Location) ([]MonthCount, error) {
	times, err := db.queries.PublishedTimes(ctx)
	if err != nil {
		return nil, fmt.Errorf("query publication times: %w", err)
	}
	return countByMonth(times, loc), nil
}

// countByMonth counts times per month in loc. The times must be sorted newest
// first; the months come in the same order.
func countByMonth(times []*time.Time, loc *time.Location) []MonthCount {
	var months []MonthCount
	for _, t := range times {
		local := t.In(loc)
		year, month := local.Year(), int(local.Month())
		if n := len(months); n > 0 && months[n-1].Year == year && months[n-1].Month == month {
			months[n-1].Count++
			continue
		}
		months = append(months, MonthCount{Year: year, Month: month, Count: 1})
	}
	return months
}

// Tags returns the tags of published posts, most used first.
func (db *DB) Tags(ctx context.Context) ([]TagCount, error) {
	tags, err := db.queries.Tags(ctx)
	if err != nil {
		return nil, fmt.Errorf("query tags: %w", err)
	}
	return tags, nil
}
