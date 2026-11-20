package db

import (
	"context"
	"os"
	"testing"
	"time"
)

// openTestDB connects to TEST_DATABASE_URL, or skips the test. The tests only
// read, so they can run against a copy of production data.
func openTestDB(t *testing.T) *DB {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL isn't set")
	}
	db, err := Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	return db
}

// TestPostsAndCountAgree checks that Posts and CountPosts, whose WHERE clauses
// are written twice in queries/posts.sql, select the same posts.
func TestPostsAndCountAgree(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	tags, err := db.Tags(ctx)
	if err != nil || len(tags) == 0 {
		t.Fatalf("no tags: %v", err)
	}
	categories, err := db.Categories(ctx)
	if err != nil || len(categories) == 0 {
		t.Fatalf("no categories: %v", err)
	}
	months, err := db.Archive(ctx)
	if err != nil || len(months) == 0 {
		t.Fatalf("no months: %v", err)
	}
	from := time.Date(months[0].Year, time.Month(months[0].Month), 1, 0, 0, 0, 0, time.UTC)

	filters := []PostFilter{
		{},
		{Category: categories[0].Slug},
		{Tag: tags[0].Name},
		{Search: "a"},
		{Search: "%"},
		{From: from, Until: from.AddDate(0, 1, 0)},
		{Category: categories[0].Slug, Search: "e"},
		{Tag: "no such tag"},
	}

	for _, filter := range filters {
		posts, err := db.Posts(ctx, filter, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		count, err := db.CountPosts(ctx, filter)
		if err != nil {
			t.Fatal(err)
		}
		if len(posts) != count {
			t.Errorf("%+v: Posts returned %d, CountPosts %d", filter, len(posts), count)
		}
	}

	if tagged, _ := db.CountPosts(ctx, PostFilter{Tag: tags[0].Name}); tagged != tags[0].Count {
		t.Errorf("tag %s: %d posts, Tags says %d", tags[0].Name, tagged, tags[0].Count)
	}
}

func TestPostsPagination(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	all, err := db.Posts(ctx, PostFilter{}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) < 3 {
		t.Skip("not enough posts")
	}

	page, err := db.Posts(ctx, PostFilter{}, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 || page[0].ID != all[1].ID || page[1].ID != all[2].ID {
		t.Errorf("limit 2 offset 1 returned the wrong posts")
	}

	post, err := db.PostByNumber(ctx, all[0].Number)
	if err != nil || post == nil || post.ID != all[0].ID {
		t.Errorf("PostByNumber(%d) = %v, %v", all[0].Number, post, err)
	}
	for _, number := range []int{-1, 0} {
		if post, err := db.PostByNumber(ctx, number); (post != nil && post.Number != number) || err != nil {
			t.Errorf("PostByNumber(%d) = %v, %v", number, post, err)
		}
	}
}

func TestSearchEscapesWildcards(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	all, _ := db.CountPosts(ctx, PostFilter{})
	for _, wildcard := range []string{"%", "_"} {
		matches, err := db.CountPosts(ctx, PostFilter{Search: wildcard})
		if err != nil {
			t.Fatal(err)
		}
		if matches == all && all > 0 {
			t.Errorf("searching %q matched every post", wildcard)
		}
	}
}
